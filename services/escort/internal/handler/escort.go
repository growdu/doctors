// Package handler 把 EscortService 暴露为 REST 接口。
//
// 设计要点：
//   - 陪诊师子路由 /escorts/me/*：JWT uid = escort 自己的 user_id；不允许改他人。
//   - /escorts/:id：陪诊师公开详情（v1 不接；保留 SetAvailability / Get 仅用于兼容）。
//   - locations（/me/location, /me/city）：只允许 owner。
//   - qualifications / trainings：CRUD。
//   - availability（v1.1）：挂载 availability 子包 handler。
package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/escort/internal/middleware"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的 service 接口。
//
// 注入接口便于单测 fake；保持与 admin-service 风格一致。
type Service interface {
	Register(ctx context.Context, userID int64) (*service.Escort, error)
	Get(ctx context.Context, id int64) (*service.Escort, error)
	SetAvailability(ctx context.Context, id int64, available bool, validUntil time.Time) error
	UpdateLocation(ctx context.Context, callerID int64, lat, lng float64) error
	UpdateCity(ctx context.Context, callerID int64, city string) error

	CreateQualification(ctx context.Context, callerID int64, q *service.Qualification) (*service.Qualification, error)
	ListQualifications(ctx context.Context, callerID int64) ([]*service.Qualification, error)
	UpdateQualification(ctx context.Context, callerID, qid int64, patch map[string]any) (*service.Qualification, error)
	DeleteQualification(ctx context.Context, callerID, qid int64) error

	CreateTraining(ctx context.Context, callerID int64, t *service.Training) (*service.Training, error)
	ListTrainings(ctx context.Context, callerID int64) ([]*service.Training, error)
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造 Handler。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 escort 路由挂到 RouterGroup。
//
// auth 中间件已先挂在 v1 上；handler 内部取 uid。
func (h *Handler) RegisterRoutes(v1 gin.IRouter) {
	e := v1.Group("/escorts")

	// 自我注册（role=escort）
	e.POST("", h.Register)
	// 公开：按 id 查 escort（v1 简化）
	e.GET("/:id", h.Get)
	// 兼容旧路径（admin 调或内部调用）：按 id 设可用状态
	e.POST("/:id/availability", h.SetAvailability)

	// /me/*：陪诊师自己（owner 校验在 service 里）
	me := e.Group("/me")
	me.PATCH("/location", h.UpdateLocation)
	me.PATCH("/city", h.UpdateCity)

	// qualifications CRUD
	q := me.Group("/qualifications")
	q.POST("", h.CreateQualification)
	q.GET("", h.ListQualifications)
	q.PATCH("/:id", h.UpdateQualification)
	q.DELETE("/:id", h.DeleteQualification)

	// trainings
	tr := me.Group("/trainings")
	tr.POST("", h.CreateTraining)
	tr.GET("", h.ListTrainings)
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
	if v, ok := c.Get(middleware.UserIDKey); ok {
		if uid, ok := v.(int64); ok {
			return uid
		}
	}
	return 0
}

// ---------- 公开 / 兼容旧路径 ----------

// Register POST /api/v1/escorts（陪诊师自我注册，role=escort）。
func (h *Handler) Register(c *gin.Context) {
	uid := uidFromCtx(c)
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

// Get GET /api/v1/escorts/:id（公开）。
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
	Available   bool   `json:"available"`
	ValidUntilRFC string `json:"valid_until,omitempty"`
}

// SetAvailability POST /api/v1/escorts/:id/availability（兼容旧路径）。
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

// ---------- /me/location, /me/city ----------

type locationReq struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

// UpdateLocation PATCH /api/v1/escorts/me/location。
func (h *Handler) UpdateLocation(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req locationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateLocation(c.Request.Context(), uid, req.Lat, req.Lng); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

type cityReq struct {
	City string `json:"city" binding:"required"`
}

// UpdateCity PATCH /api/v1/escorts/me/city。
func (h *Handler) UpdateCity(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req cityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if err := h.svc.UpdateCity(c.Request.Context(), uid, req.City); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

// ---------- /me/qualifications ----------

type createQualReq struct {
	Type      string    `json:"type" binding:"required"`
	Number    string    `json:"number" binding:"required"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	ImageURL  string    `json:"image_url"`
}

// CreateQualification POST /api/v1/escorts/me/qualifications。
func (h *Handler) CreateQualification(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req createQualReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	q, err := h.svc.CreateQualification(c.Request.Context(), uid, &service.Qualification{
		Type:      req.Type,
		Number:    req.Number,
		IssuedAt:  req.IssuedAt,
		ExpiresAt: req.ExpiresAt,
		ImageURL:  req.ImageURL,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, qualificationView(q))
}

// ListQualifications GET /api/v1/escorts/me/qualifications。
func (h *Handler) ListQualifications(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.ListQualifications(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, q := range list {
		out = append(out, qualificationView(q))
	}
	httpx.OK(c, gin.H{"items": out})
}

type updateQualReq struct {
	Type      *string    `json:"type"`
	Number    *string    `json:"number"`
	ImageURL  *string    `json:"image_url"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// UpdateQualification PATCH /api/v1/escorts/me/qualifications/:id。
func (h *Handler) UpdateQualification(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateQualReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	patch := map[string]any{}
	if req.Type != nil {
		patch["type"] = *req.Type
	}
	if req.Number != nil {
		patch["number"] = *req.Number
	}
	if req.ImageURL != nil {
		patch["image_url"] = *req.ImageURL
	}
	if req.ExpiresAt != nil {
		patch["expires_at"] = *req.ExpiresAt
	}
	q, err := h.svc.UpdateQualification(c.Request.Context(), uid, id, patch)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, qualificationView(q))
}

// DeleteQualification DELETE /api/v1/escorts/me/qualifications/:id。
func (h *Handler) DeleteQualification(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteQualification(c.Request.Context(), uid, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, nil)
}

func qualificationView(q *service.Qualification) gin.H {
	if q == nil {
		return nil
	}
	return gin.H{
		"id":         q.ID,
		"escort_id":  q.EscortID,
		"type":       q.Type,
		"number":     q.Number,
		"issued_at":  q.IssuedAt,
		"expires_at": q.ExpiresAt,
		"image_url":  q.ImageURL,
		"verified":   q.Verified,
		"created_at": q.CreatedAt,
	}
}

// ---------- /me/trainings ----------

type createTrainingReq struct {
	Title          string    `json:"title" binding:"required"`
	Provider       string    `json:"provider"`
	CompletedAt    time.Time `json:"completed_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	CertificateURL string    `json:"certificate_url"`
}

// CreateTraining POST /api/v1/escorts/me/trainings。
func (h *Handler) CreateTraining(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	var req createTrainingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	t, err := h.svc.CreateTraining(c.Request.Context(), uid, &service.Training{
		Title:          req.Title,
		Provider:       req.Provider,
		CompletedAt:    req.CompletedAt,
		ExpiresAt:      req.ExpiresAt,
		CertificateURL: req.CertificateURL,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, trainingView(t))
}

// ListTrainings GET /api/v1/escorts/me/trainings。
func (h *Handler) ListTrainings(c *gin.Context) {
	uid := uidFromCtx(c)
	if uid == 0 {
		respondError(c, errs.New(errs.CodeUnauthorized, "no user"))
		return
	}
	list, err := h.svc.ListTrainings(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, trainingView(t))
	}
	httpx.OK(c, gin.H{"items": out})
}

func trainingView(t *service.Training) gin.H {
	if t == nil {
		return nil
	}
	return gin.H{
		"id":              t.ID,
		"escort_id":       t.EscortID,
		"title":           t.Title,
		"provider":        t.Provider,
		"completed_at":    t.CompletedAt,
		"expires_at":      t.ExpiresAt,
		"certificate_url": t.CertificateURL,
		"created_at":      t.CreatedAt,
	}
}

// 兜底：errors 包引入使用
var _ = errors.New
