// Package server 启动 wallet-service HTTP server。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Server HTTP server 包装。
type Server struct {
	addr string
	srv  *http.Server
}

// New 构造 server。
func New(addr string, h http.Handler) *Server {
	return &Server{
		addr: addr,
		srv: &http.Server{
			Addr:              addr,
			Handler:           h,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}
}

// Run 启动 + 优雅停机。
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("wallet: shutdown: %w", err)
		}
		return nil
	}
}