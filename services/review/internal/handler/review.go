// Package handler 翻译 review HTTP 请求 ↔ service 调用 + errs 业务码。
//
// 设计要点：
//   - 3 个 API（create / list / detail） + 1 个 reply（admin）。
//   - 全部要求 JWT（Auth 中间件已在 router 挂载）。
//   - errors.As 优先翻译为 errs.Error，其它走 CodeInternal。
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的 service 接口。
type Service interface {
	CreateReview(ctx context.Context, reviewerID, orderID, escortID int64, rating int, comment string) (*service.Review, error)
	GetByID(ctx context.Context, id int64) (*service.Review, error)
	List(ctx context.Context, f service.ListFilter) ([]*service.Review, error)
	Reply(ctx context.Context, id, adminID int64, body string) (*service.Review, error)
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 review 4 个 API 挂到 v1 group。
//
// Auth 中间件已先挂在 v1 上。
func (h *Handler) RegisterRoutes(v1 gin.IRouter) {
	r := v1.Group("/reviews")
	r.POST("", h.Create)
	r.GET("", h.List)
	r.GET("/:id", h.Detail)
	r.POST("/:id/reply", h.Reply)
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

// createReq 是 POST /api/v1/reviews 的请求体。
type createReq struct {
	OrderID  int64  `json:"order_id" binding:"required"`
	EscortID int64  `json:"escort_id" binding:"required"`
	Rating   int    `json:"rating" binding:"required"`
	Comment  string `json:"comment"`
}

// Create POST /api/v1/reviews。
func (h *Handler) Create(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	r, err := h.svc.CreateReview(c.Request.Context(), uid, req.OrderID, req.EscortID, req.Rating, req.Comment)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         r.ID,
		"order_id":   r.OrderID,
		"escort_id":  r.EscortID,
		"rating":     r.Rating,
		"comment":    r.Comment,
		"created_at": r.CreatedAt,
	})
}

// List GET /api/v1/reviews?escort_id=&order_id=&min_rating=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	escortID, _ := strconv.ParseInt(c.Query("escort_id"), 10, 64)
	orderID, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
	minRating, _ := strconv.Atoi(c.Query("min_rating"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, err := h.svc.List(c.Request.Context(), service.ListFilter{
		EscortID:  escortID,
		OrderID:   orderID,
		MinRating: minRating,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"reviews": list, "page": page, "page_size": pageSize})
}

// Detail GET /api/v1/reviews/:id
func (h *Handler) Detail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	r, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":          r.ID,
		"order_id":    r.OrderID,
		"escort_id":   r.EscortID,
		"reviewer_id": r.ReviewerID,
		"rating":      r.Rating,
		"comment":     r.Comment,
		"reply":       r.Reply,
		"replied_by":  r.RepliedBy,
		"created_at":  r.CreatedAt,
	})
}

// replyReq 是 POST /api/v1/reviews/:id/reply 的请求体。
type replyReq struct {
	Body string `json:"body" binding:"required"`
}

// Reply POST /api/v1/reviews/:id/reply（admin）。
func (h *Handler) Reply(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req replyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	r, err := h.svc.Reply(c.Request.Context(), id, uid, req.Body)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         r.ID,
		"reply":      r.Reply,
		"replied_by": r.RepliedBy,
	})
}
