// Package handler 翻译 sos HTTP 请求 ↔ service 调用 + errs 业务码。
//
// 设计要点：
//   - 4 个 API 全部要求 JWT（Auth 中间件已在 router 挂载）。
//   - errors.As 优先翻译为 errs.Error，其它走 CodeInternal。
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/sos/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的 service 接口。
type Service interface {
	Raise(ctx context.Context, orderID, userID int64, lat, lng float64, note string) (*service.SOS, error)
	GetByID(ctx context.Context, id int64) (*service.SOS, error)
	List(ctx context.Context, f service.ListFilter) ([]*service.SOS, error)
	Resolve(ctx context.Context, sosID int64) error
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 sos 4 个 API 挂到 v1 group。
//
// Auth 中间件已先挂在 v1 上。
func (h *Handler) RegisterRoutes(v1 gin.IRouter) {
	s := v1.Group("/sos")
	s.POST("", h.Raise)
	s.GET("", h.List)
	s.GET("/:id", h.Detail)
	s.POST("/:id/resolve", h.Resolve)
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

// raiseReq 是 POST /api/v1/sos 的请求体。
type raiseReq struct {
	OrderID int64   `json:"order_id" binding:"required"`
	Lat     float64 `json:"lat" binding:"required"`
	Lng     float64 `json:"lng" binding:"required"`
	Note    string  `json:"note"`
}

// Raise POST /api/v1/sos。
func (h *Handler) Raise(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req raiseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	sos, err := h.svc.Raise(c.Request.Context(), req.OrderID, uid, req.Lat, req.Lng, req.Note)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         sos.ID,
		"order_id":   sos.OrderID,
		"user_id":    sos.UserID,
		"lat":        sos.Lat,
		"lng":        sos.Lng,
		"note":       sos.Note,
		"status":     sos.Status,
		"raised_at":  sos.RaisedAt,
	})
}

// List GET /api/v1/sos?order_id=&status=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	orderID, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, err := h.svc.List(c.Request.Context(), service.ListFilter{
		OrderID:  orderID,
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"sos": list, "page": page, "page_size": pageSize})
}

// Detail GET /api/v1/sos/:id
func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	sos, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":          sos.ID,
		"order_id":    sos.OrderID,
		"user_id":     sos.UserID,
		"lat":         sos.Lat,
		"lng":         sos.Lng,
		"note":        sos.Note,
		"status":      sos.Status,
		"raised_at":   sos.RaisedAt,
		"resolved_at": sos.ResolvedAt,
	})
}

// Resolve POST /api/v1/sos/:id/resolve（admin）。
func (h *Handler) Resolve(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Resolve(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id, "status": "resolved"})
}
