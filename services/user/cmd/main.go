// user-service 入口。
//
// 阶段：装配 server + 假 repo；接 DB 后替换 nilRepo。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/hospital"
	pkgpkg "github.com/growdu/doctors/services/user/internal/pkg"
	"github.com/growdu/doctors/services/user/internal/server"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/services/user/internal/virtualnumber"
	"github.com/growdu/doctors/shared/config"
	"github.com/growdu/doctors/shared/logger"
)

func main() {
	cfg, err := config.Load("user")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.SetLevel(parseLevel(cfg.Logging.Level))
	defer func() { _ = logger.L().Sync() }()

	profileSvc := service.New(nilProfileRepo{})
	addrSvc := address.NewService(nilAddrRepo{})
	couponSvc := coupon.NewService(nilCouponRepo{})
	hospitalSvc := hospital.NewService(nilHospitalRepo{})
	pkgSvc := pkgpkg.NewService(nilPackageRepo{})
	vnSvc := virtualnumber.NewService(nilVNRepo{})
	h := handler.New(profileSvc)
	srv := server.New(cfg.HTTP.Addr, h, cfg.Auth.JWTSecret, addrSvc, couponSvc, hospitalSvc, pkgSvc, vnSvc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.L().Info("user-service starting", zap.String("addr", cfg.HTTP.Addr))
	if err := srv.Run(ctx); err != nil {
		logger.L().Error("user-service exited", zap.Error(err))
		os.Exit(1)
	}
	logger.L().Info("user-service stopped")
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// nilProfileRepo 占位实现：任何调用都返回错误，便于 dev 期间发现。
type nilProfileRepo struct{}

func (nilProfileRepo) GetByID(ctx context.Context, id int64) (*service.Profile, error) {
	return nil, errNil
}
func (nilProfileRepo) UpdateNickname(ctx context.Context, id int64, n string) error { return errNil }
func (nilProfileRepo) UpdateAvatar(ctx context.Context, id int64, u string) error  { return errNil }

// nilAddrRepo 占位实现：address 模块的假 repo。
type nilAddrRepo struct{}

func (nilAddrRepo) Create(ctx context.Context, a *address.Record) error             { return errNil }
func (nilAddrRepo) CreateDefault(ctx context.Context, a *address.Record) error       { return errNil }
func (nilAddrRepo) ListByUser(ctx context.Context, userID int64) ([]*address.Record, error) { return nil, errNil }
func (nilAddrRepo) CountByUser(ctx context.Context, userID int64) (int, error)      { return 0, errNil }
func (nilAddrRepo) GetByID(ctx context.Context, id, userID int64) (*address.Record, error) {
	return nil, errNil
}
func (nilAddrRepo) Update(ctx context.Context, a *address.Record) error             { return errNil }
func (nilAddrRepo) SetDefault(ctx context.Context, id, userID int64) error          { return errNil }
func (nilAddrRepo) Delete(ctx context.Context, id, userID int64) error              { return errNil }

// nilCouponRepo 占位实现：coupon 模块的假 repo。
type nilCouponRepo struct{}

func (nilCouponRepo) ListActive(ctx context.Context, limit, offset int) ([]*coupon.Record, error) {
	return nil, errNil
}
func (nilCouponRepo) GetByID(ctx context.Context, id int64) (*coupon.Record, error) {
	return nil, errNil
}
func (nilCouponRepo) Claim(ctx context.Context, userID, couponID int64) (*coupon.UserCoupon, error) {
	return nil, errNil
}
func (nilCouponRepo) ListByUser(ctx context.Context, userID int64) ([]*coupon.UserCouponWithTemplate, error) {
	return nil, errNil
}
func (nilCouponRepo) MarkUsed(ctx context.Context, id, userID int64) error { return errNil }

// nilHospitalRepo 占位实现：hospital 模块的假 repo。
type nilHospitalRepo struct{}

func (nilHospitalRepo) List(ctx context.Context, f hospital.ListFilter) ([]*hospital.Record, int, error) {
	return nil, 0, errNil
}
func (nilHospitalRepo) GetByID(ctx context.Context, id int64) (*hospital.Record, error) { return nil, errNil }

// nilPackageRepo 占位实现：pkg 模块的假 repo。
type nilPackageRepo struct{}

func (nilPackageRepo) ListByHospital(ctx context.Context, hospitalID int64) ([]*pkgpkg.Record, error) {
	return nil, errNil
}
func (nilPackageRepo) GetByID(ctx context.Context, id int64) (*pkgpkg.Record, error) { return nil, errNil }

// nilVNRepo 占位实现：virtualnumber 模块的假 repo。
type nilVNRepo struct{}

func (nilVNRepo) Allocate(ctx context.Context, in virtualnumber.AllocateInput) (*virtualnumber.Record, error) {
	return nil, errNil
}
func (nilVNRepo) GetByID(ctx context.Context, id int64) (*virtualnumber.Record, error) { return nil, errNil }
func (nilVNRepo) Release(ctx context.Context, id int64, reason string) (*virtualnumber.Record, error) {
	return nil, errNil
}

var errNil = errors.New("user: repo not wired (接 pgxpool 后替换)")