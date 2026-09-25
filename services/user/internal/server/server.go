// Package server 负责 user-service 启停。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/router"
)

type Server struct {
	httpSrv *http.Server
}

func New(addr string, h *handler.Handler, jwtSecret string, addressSvc *address.Service, couponSvc *coupon.Service) *Server {
	engine := router.New(router.Deps{AddressSvc: addressSvc, CouponSvc: couponSvc}, h, jwtSecret)
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

func (s *Server) Engine() http.Handler { return s.httpSrv.Handler }

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
			return fmt.Errorf("user: shutdown: %w", err)
		}
		return nil
	}
}