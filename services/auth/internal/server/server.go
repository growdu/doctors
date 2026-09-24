// Package server 负责启动与停止 auth-service HTTP 服务。
//
// 设计要点：
//   - 构造时只做依赖装配，不读配置；启动时由 main 调 Run。
//   - Run 内部用 http.Server 包装 gin，支持优雅停机。
//   - 后续阶段会在 NewServer 中注入 DB / Redis / Kafka / 各业务 service。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/auth/internal/router"
)

// Server 封装 http.Server，便于统一启停。
type Server struct {
	httpSrv *http.Server
}

// New 创建 Server；addr 是监听地址（如 ":8080"）。
func New(addr string) *Server {
	engine := router.New()
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

// Engine 暴露内部 gin engine，仅供测试构造请求使用。
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

// ensure gin import is used by future middleware injection.
var _ = gin.New