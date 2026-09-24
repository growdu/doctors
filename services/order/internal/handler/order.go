// Package handler 把 OrderService 暴露为 REST 接口。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Handler 持有 service 引用。
type Handler struct {
	svc *service.Service
}

// New 构造 Handler。
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 把 order 路由挂到给定 RouterGroup。
// 中间件（如 Auth）已在 RouterGroup 上挂好。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	orders := r.Group("/orders")
	orders.POST("", h.Create)
	orders.GET("", h.List)
	orders.GET("/:id", h.Get)
	orders.POST("/:id/accept", h.Accept)
	orders.POST("/:id/cancel", h.Cancel)
	orders.POST("/:id/finish", h.Finish)
}

// respondError 统一错误翻译。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// ---------- handlers ----------

// Create POST /api/v1/orders
func (h *Handler) Create(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.create: not wired (阶段 3.5)"))
}

// List GET /api/v1/orders
func (h *Handler) List(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.list: not wired (阶段 3.5)"))
}

// Get GET /api/v1/orders/{id}
func (h *Handler) Get(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.get: not wired (阶段 3.5)"))
}

// Accept POST /api/v1/orders/{id}/accept
func (h *Handler) Accept(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.accept: not wired (阶段 3.6)"))
}

// Cancel POST /api/v1/orders/{id}/cancel
func (h *Handler) Cancel(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.cancel: not wired (阶段 3.5)"))
}

// Finish POST /api/v1/orders/{id}/finish
func (h *Handler) Finish(c *gin.Context) {
	respondError(c, errs.New(errs.CodeInternal, "order.finish: not wired (阶段 3.5)"))
}