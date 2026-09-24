// Package handler 把 EscortService 暴露为 REST 接口。
package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Handler 持有 service 引用。
type Handler struct {
	svc *service.Service
}

// New 构造 Handler。
func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 escort 路由挂到 RouterGroup。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	e := r.Group("/escorts")
	e.POST("", h.Register)
	e.GET("/:id", h.Get)
	e.POST("/:id/availability", h.SetAvailability)
	e.PATCH("/:id/location", h.UpdateLocation)
	e.PATCH("/:id/city", h.UpdateCity)
}

func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

func parseID(c *gin.Context) (int64, bool) {
	var id int64
	for _, ch := range c.Param("id") {
		if ch < '0' || ch > '9' {
			respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
			return 0, false
		}
		id = id*10 + int64(ch-'0')
	}
	return id, true
}

// Register POST /api/v1/escorts
func (h *Handler) Register(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can register"))
		return
	}
	e, err := h.svc.Register(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":      e.ID,
		"user_id": e.UserID,
		"status":  e.Status,
	})
}

// Get GET /api/v1/escorts/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	e, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":              e.ID,
		"user_id":         e.UserID,
		"city":            e.City,
		"status":          e.Status,
		"rating":          e.Rating,
		"available_from":  e.AvailableFrom,
		"available_until": e.AvailableUntil,
	})
}

type availabilityReq struct {
	Available  bool   `json:"available"`
	ValidUntilRFC string `json:"valid_until,omitempty"`
}

// SetAvailability POST /api/v1/escorts/:id/availability
func (h *Handler) SetAvailability(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req availabilityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	var until time.Time
	if req.ValidUntilRFC != "" {
		u, err := time.Parse(time.RFC3339, req.ValidUntilRFC)
		if err != nil {
			respondError(c, errs.New(errs.CodeParamInvalid, "invalid valid_until"))
			return
		}
		until = u
	}
	if err := h.svc.SetAvailability(c.Request.Context(), id, req.Available, until); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

type locationReq struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

// UpdateLocation PATCH /api/v1/escorts/:id/location
func (h *Handler) UpdateLocation(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req locationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateLocation(c.Request.Context(), id, uid, req.Lat, req.Lng); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

type cityReq struct {
	City string `json:"city" binding:"required"`
}

// UpdateCity PATCH /api/v1/escorts/:id/city
func (h *Handler) UpdateCity(c *gin.Context) {
	uid := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req cityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateCity(c.Request.Context(), id, uid, req.City); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// Role 是 middleware 的 Role helper；handler 上一行已被 middleware 导出。
var _ = middleware.Role