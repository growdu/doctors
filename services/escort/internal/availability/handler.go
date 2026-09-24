package availability

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Handler 暴露 escort_availabilities 的 4 个 REST API（v1.1 escort-availability plan）。
type Handler struct {
	svc *Service
}

// NewHandler 构造 handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 availability 路由挂到 RouterGroup。
//
// 路由（r 是 authed group；公共端点 ListByEscort 由 RegisterPublicRoutes 挂到无 auth 的 group）：
//   - PUT    /api/v1/escorts/me/availability         加新时段（自己）
//   - DELETE /api/v1/escorts/me/availability/:id     删时段（自己）
//   - GET    /api/v1/escorts/me/availability         查自己的所有时段
//   - GET    /api/v1/escorts/:id/availabilities      公开：查 escort 在某时段窗口内的可用时段（由 RegisterPublicRoutes 挂）
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	me := r.Group("/escorts/me/availability")
	me.PUT("", h.AddMine)
	me.DELETE("/:id", h.RemoveMine)
	me.GET("", h.ListMine)
}

// RegisterPublicRoutes 把公开路由挂到无 auth 的 RouterGroup。
func (h *Handler) RegisterPublicRoutes(r gin.IRouter) {
	r.GET("/escorts/:id/availabilities", h.ListByEscort)
}

// ---------- /escorts/me/availability ----------

type addReq struct {
	StartAt string `json:"start_at" binding:"required"` // RFC3339
	EndAt   string `json:"end_at" binding:"required"`
}

// AddMine PUT /api/v1/escorts/me/availability。
func (h *Handler) AddMine(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	if role := middleware.Role(c); role != "escort" {
		respondError(c, errs.New(errs.CodeForbidden, "only escort can add availability"))
		return
	}
	var req addReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	start, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid start_at"))
		return
	}
	end, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid end_at"))
		return
	}
	a, err := h.svc.AddAvailability(c.Request.Context(), uid, uid, start, end)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         a.ID,
		"escort_id":  a.EscortID,
		"start_at":   a.StartAt,
		"end_at":     a.EndAt,
		"status":     a.Status,
	})
}

// RemoveMine DELETE /api/v1/escorts/me/availability/:id。
func (h *Handler) RemoveMine(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.svc.RemoveAvailability(c.Request.Context(), id, uid); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// ListMine GET /api/v1/escorts/me/availability。
func (h *Handler) ListMine(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.ListMyAvailabilities(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": list})
}

// ---------- /escorts/:id/availabilities（公开） ----------

// ListByEscort GET /api/v1/escorts/:id/availabilities?start_at=&end_at=&limit=
//
// 公开端点（无需 auth）：返回 escort 在 [start_at, end_at) 内的 available 时段。
// 用于 patient-miniapp 渲染"该 escort 在 X 时段可服务"。
func (h *Handler) ListByEscort(c *gin.Context) {
	escortID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	startStr := c.Query("start_at")
	endStr := c.Query("end_at")
	if startStr == "" || endStr == "" {
		respondError(c, errs.New(errs.CodeParamInvalid, "start_at and end_at required"))
		return
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid start_at"))
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid end_at"))
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	list, err := h.svc.ListEscortAvailabilities(c.Request.Context(), escortID, start, end, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": list})
}

// respondError 统一错误翻译：先看 *Error（errs.As），再看 sentinel（errors.Is），最后 internal。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	switch {
	case errors.Is(err, ErrAvailabilityNotFound):
		httpx.Fail(c, int(errs.CodeNotFound), err.Error())
	case errors.Is(err, ErrAvailabilityConflict), errors.Is(err, ErrTimeInvalid):
		httpx.Fail(c, int(errs.CodeUnprocessable), err.Error())
	case errors.Is(err, ErrNotOwner):
		httpx.Fail(c, int(errs.CodeForbidden), err.Error())
	case errors.Is(err, ErrBookedAlready), errors.Is(err, ErrNotBookable):
		httpx.Fail(c, int(errs.CodeConflict), err.Error())
	default:
		httpx.Fail(c, int(errs.CodeInternal), err.Error())
	}
}