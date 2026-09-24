// Package handler 把 Service 暴露为 REST 接口。
//
// 设计要点：
//   - 每个 handler 只做参数解析 → 调 service → 用 httpx.respond 写回。
//   - errs.As 翻译业务错误：code 不等于 0 → httpx.Fail。
//   - 不在这里写业务规则；那是 service 的事。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/auth/internal/middleware"
	"github.com/growdu/doctors/services/auth/internal/service"
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

// RegisterRoutes 把所有 auth/users 路由挂到给定 gin.IRouter 上。
// jwtSecret 用于 /me 与 /real-name/auth 鉴权中间件。
func (h *Handler) RegisterRoutes(r gin.IRouter, jwtSecret string) {
	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/sms/send", h.SendSMS)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)

	users := v1.Group("/users", middleware.Auth(jwtSecret))
	users.POST("/real-name/auth", h.RealNameAuth)
	users.GET("/me", h.Me)
}

// respondError 把业务错误翻译成 httpx.Fail；非业务错误统一 500。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// ---------- handlers ----------

type sendSMSReq struct {
	Phone string `json:"phone" binding:"required"`
}

// SendSMS POST /api/v1/auth/sms/send
func (h *Handler) SendSMS(c *gin.Context) {
	var req sendSMSReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.SendSMS(c.Request.Context(), req.Phone); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

type loginReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
	Type  string `json:"type"` // "sms" | "wx"
	WX    string `json:"wx_code,omitempty"`
}

// Login POST /api/v1/auth/login
//   - type=sms → phone + code
//   - type=wx  → wx_code
func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	var (
		tok string
		uid int64
		err error
	)
	switch req.Type {
	case "sms":
		tok, uid, err = h.svc.LoginBySMS(c.Request.Context(), req.Phone, req.Code)
	case "wx":
		tok, uid, err = h.svc.LoginByWX(c.Request.Context(), req.WX)
	default:
		respondError(c, errs.New(errs.CodeParamInvalid, "type must be sms or wx"))
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"token": tok, "user_id": uid})
}

type refreshReq struct {
	Token string `json:"token" binding:"required"`
}

// Refresh POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	tok, err := h.svc.Refresh(c.Request.Context(), req.Token)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"token": tok})
}

type realNameReq struct {
	Name   string `json:"name" binding:"required"`
	IDCard string `json:"id_card" binding:"required"`
}

// RealNameAuth POST /api/v1/users/real-name/auth
func (h *Handler) RealNameAuth(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user in ctx"))
		return
	}
	var req realNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.RealNameAuth(c.Request.Context(), uid, req.Name, req.IDCard); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// Me GET /api/v1/users/me
func (h *Handler) Me(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user in ctx"))
		return
	}
	u, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":                 u.ID,
		"phone":              u.Phone,
		"role":               u.Role,
		"real_name_verified": u.RealNameVerified,
	})
}