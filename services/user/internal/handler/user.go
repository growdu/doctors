// Package handler 把 UserService 暴露为 REST 接口。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/user/internal/middleware"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Handler 持有 service 引用。
type Handler struct {
	svc *service.Service
}

// New 构造 Handler。
func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 user 路由挂到 RouterGroup（Auth 已挂）。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	users := r.Group("/users")
	users.GET("/:id", h.Get)
	users.PATCH("/:id/nickname", h.UpdateNickname)
	users.PATCH("/:id/avatar", h.UpdateAvatar)
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

// Get GET /api/v1/users/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.GetProfile(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":                 p.ID,
		"phone":              p.Phone,
		"role":               p.Role,
		"nickname":           p.Nickname,
		"avatar_url":         p.AvatarURL,
		"real_name_verified": p.RealNameVerified,
	})
}

type updateNickReq struct {
	Nickname string `json:"nickname" binding:"required"`
}

// UpdateNickname PATCH /api/v1/users/:id/nickname
func (h *Handler) UpdateNickname(c *gin.Context) {
	caller := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateNickReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateNickname(c.Request.Context(), caller, id, req.Nickname); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

type updateAvatarReq struct {
	AvatarURL string `json:"avatar_url" binding:"required"`
}

// UpdateAvatar PATCH /api/v1/users/:id/avatar
func (h *Handler) UpdateAvatar(c *gin.Context) {
	caller := middleware.UserID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateAvatar(c.Request.Context(), caller, id, req.AvatarURL); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}