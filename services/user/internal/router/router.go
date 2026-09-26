// Package router 注册 user-service 路由。
//
// 设计要点：
//   - /healthz 不挂任何中间件（K8s liveness）。
//   - /metrics 挂 Prometheus 抓取端（promhttp.Handler()）。
//   - /api/v1/* 全部挂 Auth（JWT）。
//
// 全局中间件挂载顺序（外 → 内）：
//
//	Metrics()        最外层：401/403/panic/429 也被埋点
//	Recovery()       panic 恢复 → 500 + 业务码 500000
//	RateLimit(...)   按 IP token bucket：超限 429 + 业务码 13001
//	Auth(...)        /api/v1 group 内挂，未通过 token 不消耗限流桶
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/hospital"
	mw "github.com/growdu/doctors/services/user/internal/middleware"
	pkgpkg "github.com/growdu/doctors/services/user/internal/pkg"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/services/user/internal/virtualnumber"
	"github.com/growdu/doctors/shared/health"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/metrics"
	sharedmw "github.com/growdu/doctors/shared/middleware"
)

// Deps 装配 router 所需的依赖。
type Deps struct {
	ProfileSvc       *service.Service
	AddressSvc       *address.Service
	CouponSvc        *coupon.Service
	HospitalSvc      *hospital.Service
	PackageSvc       *pkgpkg.Service
	VirtualNumberSvc *virtualnumber.Service
}

// New 返回挂好路由的 gin engine。
// readyzM 用于 /readyz 端点（K8s readinessProbe）；传 nil 时 /readyz 永远 503（fail-closed）。
func New(d Deps, h *handler.Handler, jwtSecret string, readyzM *health.Manager) *gin.Engine {
	r := gin.New()
	// 中间件顺序：Metrics → Recovery → RateLimit（全局）
	r.Use(sharedmw.Metrics())
	r.Use(sharedmw.Recovery())
	r.Use(sharedmw.RateLimit())
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	// readiness 探活：所有依赖（DB / Kafka）都 OK 才 200
	r.GET("/readyz", health.ReadyzHandler(readyzM))
	// Prometheus 抓取端
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	v1 := r.Group("/api/v1", mw.Auth(jwtSecret))
	h.RegisterRoutes(v1)

	// address 模块独立路由
	if d.AddressSvc != nil {
		addrH := address.NewHandler(d.AddressSvc)
		addrH.RegisterRoutes(v1)
	}
	// coupon 模块独立路由
	if d.CouponSvc != nil {
		couponH := coupon.NewHandler(d.CouponSvc)
		couponH.RegisterRoutes(v1)
	}
	// hospital 模块独立路由
	if d.HospitalSvc != nil {
		hh := hospital.NewHandler(d.HospitalSvc)
		hh.RegisterRoutes(v1)
	}
	// package（服务包）模块独立路由
	if d.PackageSvc != nil {
		ph := pkgpkg.NewHandler(d.PackageSvc)
		ph.RegisterRoutes(v1)
	}
	// virtual-number 模块独立路由
	if d.VirtualNumberSvc != nil {
		vh := virtualnumber.NewHandler(d.VirtualNumberSvc)
		vh.RegisterRoutes(v1)
	}
	return r
}
