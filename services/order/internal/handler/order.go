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
//
// v1.1（order-matching-redesign）：
//   - 删除 POST /orders/:id/accept（陪诊师抢单）
//   - 新增 POST /orders/:id/select-escort（患者选陪诊师）
//   - 新增 POST /orders/:id/confirm-accept（陪诊师 30s 内确认）
//   - 新增 POST /orders/:id/reject-accept（陪诊师拒接或 scheduler 超时回退）
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	orders := r.Group("/orders")
	orders.POST("", h.Create)
	orders.GET("", h.List)
	orders.GET("/:id", h.Get)
	orders.POST("/:id/select-escort", h.SelectEscort)
	orders.POST("/:id/confirm-accept", h.ConfirmAccept)
	orders.POST("/:id/reject-accept", h.RejectAccept)
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

// SelectEscort POST /api/v1/orders/{id}/select-escort（v1.1 患者选陪诊师）。
func (h *Handler) SelectEscort(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if role := middleware.Role(c); role != "patient" {
		respondError(c, errs.New(errs.CodeForbidden, "only patient can select escort"))
		return
	}
	var body struct {
		EscortID int64 `json:"escort_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid request body"))
		return
	}
	if body.EscortID == 0 {
		respondError(c, errs.New(errs.CodeParamInvalid, "escort_id required"))
		return
	}
	res, err := h.svc.SelectEscort(c.Request.Context(), id, uid, body.EscortID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"order_id":                  res.OrderID,
		"version":                   res.Version,
		"selected_escort_id":        res.SelectedEscortID,
		"escort_pending_expire_at":  res.EscortPendingExpireAt,
	})
}

// ConfirmAccept POST /api/v1/orders/{id}/confirm-accept（v1.1 陪诊师 30s 确认）。
func (h *Handler) ConfirmAccept(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can confirm accept"))
		return
	}
	o, err := h.svc.ConfirmAccept(c.Request.Context(), id, uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, o)
}

// RejectAccept POST /api/v1/orders/{id}/reject-accept（v1.1 陪诊师拒接或 scheduler 调超时回退）。
func (h *Handler) RejectAccept(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid request body"))
		return
	}
	if body.Reason == "" {
		body.Reason = "escort_declined"
	}
	if err := h.svc.RejectAccept(c.Request.Context(), id, uid, body.Reason); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
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