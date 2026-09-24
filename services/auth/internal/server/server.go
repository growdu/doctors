// Package server 负责启动与停止 auth-service HTTP 服务。
//
// 设计要点：
//   - New 接受完整依赖（handler + jwtSecret），构造时只装配不读外部资源。
//   - Run 内部用 http.Server 包装 gin，支持优雅停机（10s 超时）。
//   - 后续阶段会在 main 里注入 DB / Redis / Kafka。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/router"
)

// Server 封装 http.Server，便于统一启停。
type Server struct {
	httpSrv *http.Server
}

// New 创建 Server；addr 监听地址，h 是业务 handler，jwtSecret 用于鉴权中间件。
func New(addr string, h *handler.Handler, jwtSecret string) *Server {
	engine := router.New(h, jwtSecret)
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

// Run 阻塞直到 ctx 取消；ctx 取消后等待 in-flight 请求完成再返回。
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("auth: shutdown: %w", err)
		}
		return nil
	}
}