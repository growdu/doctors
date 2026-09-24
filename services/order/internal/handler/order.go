// Package handler 把 OrderService 暴露为 REST 接口。
//
// 设计要点：
//   - 每个 handler 做参数绑定 → 调 service → 翻译 errs → httpx 写回。
//   - 路径 :id 用 strconv 转 int64；非法 id 立刻 400。
//   - Accept 走陪诊师（role=escort）路径；Create 走患者（role=patient）路径。
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/order/internal/middleware"
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

// RegisterRoutes 把 order 路由挂到 RouterGroup。
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

// parseID 把路径 :id 解析为 int64。
func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return 0, false
	}
	return id, true
}

// ---------- handlers ----------

type createReq struct {
	HospitalID     int64     `json:"hospital_id" binding:"required"`
	PackageID      int64     `json:"package_id" binding:"required"`
	ServiceStartAt time.Time `json:"service_start_at" binding:"required"`
	Amount         float64   `json:"amount" binding:"required"`
}

// Create POST /api/v1/orders
func (h *Handler) Create(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	o, err := h.svc.Create(c.Request.Context(), service.CreateReq{
		PatientID:      uid,
		HospitalID:     req.HospitalID,
		PackageID:      req.PackageID,
		ServiceStartAt: req.ServiceStartAt,
		Amount:         req.Amount,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         o.ID,
		"order_no":   o.OrderNo,
		"status":     o.Status,
		"created_at": o.ServiceStartAt,
	})
}

// List GET /api/v1/orders
func (h *Handler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.List(c.Request.Context(), uid, 20, 0)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"orders": list})
}

// Get GET /api/v1/orders/{id}
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	o, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, o)
}

// Accept POST /api/v1/orders/{id}/accept
func (h *Handler) Accept(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can accept"))
		return
	}
	res, err := h.svc.Accept(c.Request.Context(), id, uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": res.OrderID, "version": res.Version})
}

// Cancel POST /api/v1/orders/{id}/cancel
func (h *Handler) Cancel(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.Cancel(c.Request.Context(), id, uid, body.Reason); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// Finish POST /api/v1/orders/{id}/finish
func (h *Handler) Finish(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Finish(c.Request.Context(), id, uid); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}