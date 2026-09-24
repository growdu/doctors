// Package handler 把 MatchService 暴露为 REST 接口。
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/match/internal/middleware"
	"github.com/growdu/doctors/services/match/internal/service"
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

// RegisterRoutes 把 match 路由挂到 RouterGroup（Auth 已在 group 上挂好）。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	m := r.Group("/match")
	m.GET("/feed", h.Feed)
	m.POST("/candidates", h.Candidates)
	m.POST("/dispatch", h.Dispatch)
}

func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// Feed GET /api/v1/match/feed?order_id=123&top=10
func (h *Handler) Feed(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	orderID, err := parseInt64Query(c, "order_id")
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "order_id required"))
		return
	}
	topN, _ := parseInt64Query(c, "top")
	feed, err := h.svc.Feed(c.Request.Context(), orderID, int(topN))
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": orderID, "escorts": feed})
}

// Candidates POST /api/v1/match/candidates {order_id}
func (h *Handler) Candidates(c *gin.Context) {
	var req struct {
		OrderID int64 `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	cands, err := h.svc.Candidates(c.Request.Context(), req.OrderID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": req.OrderID, "escorts": cands})
}

// Dispatch POST /internal/match/dispatch（订单服务回调 → 写抢单池）
func (h *Handler) Dispatch(c *gin.Context) {
	var req dispatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	serviceTime, _ := time.Parse(time.RFC3339, req.ServiceStartAt)
	cands, err := h.svc.Match(c.Request.Context(), service.OrderInfo{
		ID:          req.OrderID,
		City:        req.City,
		ServiceTime: serviceTime,
		Lat:         req.HospitalLat,
		Lng:         req.HospitalLng,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": req.OrderID, "candidates": cands})
}

type dispatchReq struct {
	OrderID        int64   `json:"order_id" binding:"required"`
	City           string  `json:"city"`
	ServiceStartAt string  `json:"service_start_at"`
	HospitalLat    float64 `json:"hospital_lat"`
	HospitalLng    float64 `json:"hospital_lng"`
}

// parseInt64Query 把 query string 转 int64。
func parseInt64Query(c *gin.Context, key string) (int64, error) {
	v := c.Query(key)
	if v == "" {
		return 0, nil
	}
	return strconv.ParseInt(v, 10, 64)
}