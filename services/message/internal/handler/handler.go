// Package handler 翻译 message HTTP 请求 ↔ service 调用 + errs 业务码。
//
// 设计要点：
//   - 4 个 API 全部要求 JWT（Auth 中间件已在 router 挂载）。
//   - broadcast 由 super_admin / order_admin / audit_admin 调用。
//   - errors.As 优先翻译为 errs.Error，其它走 CodeInternal。
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/message/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的 service 接口。
//
// 注入接口便于单测 fake；保持与 admin-service 风格一致。
type Service interface {
	SendMessage(ctx context.Context, orderID, fromID, toID int64, body string) (*service.Message, error)
	GetByID(ctx context.Context, id int64) (*service.Message, error)
	ListByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*service.Message, error)
	Broadcast(ctx context.Context, fromID int64, toIDs []int64, body string) (int, error)
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 message 4 个 API 挂到 v1 group。
//
// Auth 中间件已先挂在 v1 上。
func (h *Handler) RegisterRoutes(v1 gin.IRouter) {
	m := v1.Group("/messages")
	m.POST("", h.Send)
	m.GET("", h.List)
	m.GET("/:id", h.Detail)
	m.POST("/broadcast", h.Broadcast)
}

// respondError 统一错误翻译。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// parseID 把 :id 解析为 int64。
func parseID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return 0, false
	}
	return id, true
}

// uidFromCtx 提取 JWT 注入的 user_id。
func uidFromCtx(c *gin.Context) int64 {
	if v, ok := c.Get("uid"); ok {
		if uid, ok := v.(int64); ok {
			return uid
		}
	}
	return 0
}

// sendReq 是 POST /api/v1/messages 的请求体。
type sendReq struct {
	OrderID int64  `json:"order_id" binding:"required"`
	ToUserID int64 `json:"to_user_id" binding:"required"`
	Body     string `json:"body" binding:"required"`
}

// Send POST /api/v1/messages。
func (h *Handler) Send(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req sendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	m, err := h.svc.SendMessage(c.Request.Context(), req.OrderID, uid, req.ToUserID, req.Body)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         m.ID,
		"order_id":   m.OrderID,
		"from":       m.FromUserID,
		"to":         m.ToUserID,
		"body":       m.Body,
		"created_at": m.CreatedAt,
	})
}

// List GET /api/v1/messages?order_id=&from=&to=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	list, err := h.svc.ListByOrder(c.Request.Context(), orderID, pageSize, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"messages": list, "page": page, "page_size": pageSize})
}

// Detail GET /api/v1/messages/:id
func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	m, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         m.ID,
		"order_id":   m.OrderID,
		"from":       m.FromUserID,
		"to":         m.ToUserID,
		"body":       m.Body,
		"created_at": m.CreatedAt,
	})
}

// broadcastReq 是 POST /api/v1/messages/broadcast 的请求体。
type broadcastReq struct {
	ToUserIDs []int64 `json:"to_user_ids" binding:"required"`
	Body      string  `json:"body" binding:"required"`
}

// Broadcast POST /api/v1/messages/broadcast（admin）。
func (h *Handler) Broadcast(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req broadcastReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	n, err := h.svc.Broadcast(c.Request.Context(), uid, req.ToUserIDs, req.Body)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"sent": n})
}
