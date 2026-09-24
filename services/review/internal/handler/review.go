// Package handler - ReviewService REST 接口。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/review/internal/middleware"
	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/reviews")
	g.POST("", h.Create)
	g.GET("/orders/:orderID", h.GetByOrder)
	g.GET("/escorts/:escortID", h.ListByEscort)
}

func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

func parseID(c *gin.Context, key string) (int64, bool) {
	var id int64
	for _, ch := range c.Param(key) {
		if ch < '0' || ch > '9' {
			respondError(c, errs.New(errs.CodeParamInvalid, "invalid "+key))
			return 0, false
		}
		id = id*10 + int64(ch-'0')
	}
	return id, true
}

type createReq struct {
	OrderID  int64  `json:"order_id" binding:"required"`
	EscortID int64  `json:"escort_id" binding:"required"`
	Rating   int    `json:"rating" binding:"required"`
	Comment  string `json:"comment"`
}

func (h *Handler) Create(c *gin.Context) {
	uid := middleware.UserID(c)
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
	httpx.OK(c, gin.H{"id": r.ID, "order_id": r.OrderID, "rating": r.Rating})
}

func (h *Handler) GetByOrder(c *gin.Context) {
	id, ok := parseID(c, "orderID")
	if !ok {
		return
	}
	r, err := h.svc.GetByOrder(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, r)
}

func (h *Handler) ListByEscort(c *gin.Context) {
	id, ok := parseID(c, "escortID")
	if !ok {
		return
	}
	list, err := h.svc.ListByEscort(c.Request.Context(), id, 20, 0)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"reviews": list})
}
