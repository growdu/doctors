// Package server 负责 user-service 启停。
//
// 设计要点：
//   - Run 内部用 http.Server 包装 gin，支持优雅停机（15s 超时）。
//   - 支持通过 RegisterShutdownHook 注册资源释放回调，按 LIFO 顺序
//     在 HTTP Shutdown 完成后执行。
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/hospital"
	pkgpkg "github.com/growdu/doctors/services/user/internal/pkg"
	"github.com/growdu/doctors/services/user/internal/router"
	"github.com/growdu/doctors/services/user/internal/virtualnumber"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/logger"
)

const shutdownTimeout = 15 * time.Second

type shutdownHook struct {
	name string
	fn   func() error
}

type Server struct {
	httpSrv *http.Server
	mu      sync.Mutex
	hooks   []shutdownHook
}

func New(addr string, h *handler.Handler, jwtSecret string, readyzM *health.Manager, addressSvc *address.Service, couponSvc *coupon.Service, hospitalSvc *hospital.Service, packageSvc *pkgpkg.Service, virtualNumberSvc *virtualnumber.Service) *Server {
	engine := router.New(router.Deps{
		AddressSvc:       addressSvc,
		CouponSvc:        couponSvc,
		HospitalSvc:      hospitalSvc,
		PackageSvc:       packageSvc,
		VirtualNumberSvc: virtualNumberSvc,
	}, h, jwtSecret, readyzM)
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

// RegisterShutdownHook 注册资源释放回调（LIFO）。
func (s *Server) RegisterShutdownHook(name string, fn func() error) {
	if name == "" || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks = append(s.hooks, shutdownHook{name: name, fn: fn})
}

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("user: shutdown: %w", err)
		}
		s.runShutdownHooks()
		return nil
	}
}

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
