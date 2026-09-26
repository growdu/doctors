// Package server 启动 admin-service HTTP server。
//
// 设计要点：
//   - 监听失败立即返回；ctx.Done() → Shutdown(15s)。
//   - 支持通过 RegisterShutdownHook 在 HTTP Shutdown 完成后释放资源
//     （DB pool / Kafka writer / OTel tracer 等），按 LIFO 顺序。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/shared/logger"
)

// shutdownTimeout 是收到 ctx.Done() 后等待 in-flight 请求完成的最长时间。
const shutdownTimeout = 15 * time.Second

// shutdownHook 是 Run() 退出阶段按 LIFO 顺序调用的资源释放回调。
type shutdownHook struct {
	name string
	fn   func() error
}

// Server 是 HTTP server 包装。
type Server struct {
	addr  string
	srv   *http.Server
	mu    sync.Mutex
	hooks []shutdownHook
}

// New 构造。
func New(addr string, h http.Handler) *Server {
	return &Server{
		addr: addr,
		srv: &http.Server{
			Addr:              addr,
			Handler:           h,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}
}

// RegisterShutdownHook 注册资源释放回调（LIFO）。
//
// 典型使用：
//
//	srv.RegisterShutdownHook("otel-tracer", func() error {
//	    return traceShutdown(context.Background())
//	})
//	srv.RegisterShutdownHook("db-pool", pool.Close)
func (s *Server) RegisterShutdownHook(name string, fn func() error) {
	if name == "" || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = append(s.hooks, shutdownHook{name: name, fn: fn})
}

// Run 启动 + 优雅停机。
//
// 关闭顺序：ctx.Done() → Shutdown(15s) → registered hooks (LIFO)。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	logger.L().Info("admin-service starting", zap.String("addr", s.addr))

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("admin: shutdown: %w", err)
		}
		s.runShutdownHooks()
		return nil
	}
}

// runShutdownHooks 按 LIFO 顺序调用注册的 hook；任一失败只记录日志不中断。
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
						zap.String("hook", h.name), zap.Any("recover", r))
				}
			}()
			if err := h.fn(); err != nil {
				logger.L().Warn("shutdown hook returned error",
					zap.String("hook", h.name), zap.Error(err))
			}
		}()
	}
}
