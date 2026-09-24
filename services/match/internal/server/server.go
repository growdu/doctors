// Package server 负责启动与停止 match-service HTTP 服务。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/growdu/doctors/services/match/internal/handler"
	"github.com/growdu/doctors/services/match/internal/router"
)

// Server 封装 http.Server。
type Server struct {
	httpSrv *http.Server
}

// New 创建 Server。
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

// Run 阻塞直到 ctx 取消。
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
			return fmt.Errorf("match: shutdown: %w", err)
		}
		return nil
	}
}