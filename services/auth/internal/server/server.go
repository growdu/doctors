// Package server 负责启动与停止 auth-service HTTP 服务。
//
// 设计要点：
//   - New 接受完整依赖（handler + jwtSecret），构造时只装配不读外部资源。
//   - Run 内部用 http.Server 包装 gin，支持优雅停机（15s 超时）。
//   - 支持通过 RegisterShutdownHook 注册资源释放回调（DB pool / Kafka writer
//     / OTel tracer 等），按 LIFO 顺序在 HTTP Shutdown 完成后执行。
//   - 后续阶段会在 main 里注入 DB / Redis / Kafka 并通过 hook 释放。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/router"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/logger"
)

// shutdownTimeout 是收到 ctx.Done() 后等待 in-flight 请求完成的最长时间。
//
// 15s 优于 Go 标准 10s：DB pool 关闭、Kafka writer flush、OTel span
// 导出都需要时间，太短会丢数据；过长则 K8s rolling 升级时会被 SIGKILL。
const shutdownTimeout = 15 * time.Second

// shutdownHook 是 Run() 退出阶段按 LIFO 顺序调用的资源释放回调。
type shutdownHook struct {
	name string
	fn   func() error
}

// Server 封装 http.Server，便于统一启停。
type Server struct {
	httpSrv *http.Server
	mu      sync.Mutex
	hooks   []shutdownHook
}

// New 创建 Server；addr 监听地址，h 是业务 handler，jwtSecret 用于鉴权中间件，
// readyzM 用于 /readyz 端点（K8s readinessProbe）。readyzM 为 nil 时 /readyz 永远 503。
func New(addr string, h *handler.Handler, jwtSecret string, readyzM *health.Manager) *Server {
	engine := router.New(h, jwtSecret, readyzM)
	return &Server{
		httpSrv: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}
}

// Engine 暴露内部 handler，仅供测试构造请求使用。
func (s *Server) Engine() http.Handler { return s.httpSrv.Handler }

// RegisterShutdownHook 注册资源释放回调。
//
// 调用时机：HTTP Server.Shutdown 返回之后。多个 hook 按注册的逆序
// （LIFO）依次调用 —— 后注册先释放，与 defer 语义一致，便于"先关子资源、
// 再关父资源"。
//
// 典型使用：
//
//	srv.RegisterShutdownHook("otel-tracer", func() error {
//	    return traceShutdown(context.Background())
//	})
//	srv.RegisterShutdownHook("db-pool", pool.Close)
//
// 当前阶段（DB/Kafka 未接入）通常不调用；接入后再补。
func (s *Server) RegisterShutdownHook(name string, fn func() error) {
	if name == "" || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = append(s.hooks, shutdownHook{name: name, fn: fn})
}

// Run 阻塞直到 ctx 取消；ctx 取消后等待 in-flight 请求完成再返回。
//
// 关闭顺序：ctx.Done() → httpSrv.Shutdown(15s) → registered hooks (LIFO)。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("auth: shutdown: %w", err)
		}
		s.runShutdownHooks()
		return nil
	}
}

// runShutdownHooks 按 LIFO 顺序调用注册的 hook；任一失败只记录日志不中断。
//
// 把 panic 也吞掉，避免一个坏 hook 影响其他资源释放（DB pool 关闭比
// Kafka writer 更关键，不能因为 Kafka Close 偶发 panic 就拖垮整个关闭流程）。
func (s *Server) runShutdownHooks() {
	s.mu.Lock()
	hooks := append([]shutdownHook(nil), s.hooks...)
	s.mu.Unlock()

	for i := len(hooks) - 1; i >= 0; i-- {
		h := hooks[i]
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.L().Error("shutdown hook panicked",
						zap.String("hook", h.name),
						zap.Any("recover", r))
				}
			}()
			if err := h.fn(); err != nil {
				logger.L().Warn("shutdown hook returned error",
					zap.String("hook", h.name),
					zap.Error(err))
			}
		}()
	}
}
