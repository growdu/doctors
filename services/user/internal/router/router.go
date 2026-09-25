// Package router 注册 user-service 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/address"
	"github.com/growdu/doctors/services/user/internal/coupon"
	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/hospital"
	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// Deps 装配 router 所需的依赖。
type Deps struct {
	ProfileSvc *service.Service
	AddressSvc *address.Service
	CouponSvc  *coupon.Service
	HospitalSvc *hospital.Service
}

// New 返回挂好路由的 gin engine。
func New(d Deps, h *handler.Handler, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", func(c *gin.Context) {
		httpx.OK[any](c, gin.H{"status": "ok"})
	})
	v1 := r.Group("/api/v1", middleware.Auth(jwtSecret))
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
	return r
}