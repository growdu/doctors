// Package address 实现 user-service 地址簿的 service + handler。
//
// 设计要点：
//   - Service 是业务编排层：地址上限 5、参数校验、调用 Repo。
//   - Handler 把 Service 暴露为 REST（gin）。
//   - callerID 由 JWT 中间件注入；handler 不再校验 caller。
//   - Repo 已经是接口形态，业务单测用 fake repo 即可。
package address

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// MaxPerUser 是单用户地址上限。
const MaxPerUser = 5

// Repository 是 address 模块需要的仓储契约（service 单测用 fake 实现）。
type Repository interface {
	Create(ctx context.Context, a *Record) error
	CreateDefault(ctx context.Context, a *Record) error
	ListByUser(ctx context.Context, userID int64) ([]*Record, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	GetByID(ctx context.Context, id, userID int64) (*Record, error)
	Update(ctx context.Context, a *Record) error
	SetDefault(ctx context.Context, id, userID int64) error
	Delete(ctx context.Context, id, userID int64) error
}

// Service 是 address 业务编排器。
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService 装配 Service；now 默认为 time.Now，单测可注入。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now() }}
}

// ---------- service methods ----------

// List 列出 caller 的所有地址。
func (s *Service) List(ctx context.Context, userID int64) ([]*Record, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list addresses", err)
	}
	return list, nil
}

// CreateInput 是新增地址的入参（lat/lng 可选）。
type CreateInput struct {
	Recipient string   `json:"recipient"`
	Phone     string   `json:"phone"`
	Detail    string   `json:"detail"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
	IsDefault bool     `json:"is_default"`
}

// Create 新增地址；上限校验 + 参数校验。
func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (*Record, error) {
	if err := validateFields(in.Recipient, in.Phone, in.Detail); err != nil {
		return nil, err
	}
	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "count addresses", err)
	}
	if count >= MaxPerUser {
		return nil, errs.New(errs.CodeConflict, "address limit reached (max 5)")
	}
	a := &Record{
		UserID:    userID,
		Recipient: strings.TrimSpace(in.Recipient),
		Phone:     strings.TrimSpace(in.Phone),
		Detail:    strings.TrimSpace(in.Detail),
		Lat:       in.Lat,
		Lng:       in.Lng,
	}
	if in.IsDefault {
		if err := s.repo.CreateDefault(ctx, a); err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "create default address", err)
		}
	} else {
		if err := s.repo.Create(ctx, a); err != nil {
			return nil, errs.Wrap(errs.CodeInternal, "create address", err)
		}
	}
	return a, nil
}

// UpdateInput 是修改地址的入参（lat/lng 可选）。
type UpdateInput struct {
	Recipient string   `json:"recipient"`
	Phone     string   `json:"phone"`
	Detail    string   `json:"detail"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
}

// Update 修改地址；参数校验 + caller 必须是所有者。
func (s *Service) Update(ctx context.Context, callerID, id int64, in UpdateInput) (*Record, error) {
	if err := validateFields(in.Recipient, in.Phone, in.Detail); err != nil {
		return nil, err
	}
	a := &Record{
		ID:        id,
		UserID:    callerID,
		Recipient: strings.TrimSpace(in.Recipient),
		Phone:     strings.TrimSpace(in.Phone),
		Detail:    strings.TrimSpace(in.Detail),
		Lat:       in.Lat,
		Lng:       in.Lng,
	}
	if err := s.repo.Update(ctx, a); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "address not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "update address", err)
	}
	return a, nil
}

// SetDefault 把 id 设为 caller 的默认地址。
func (s *Service) SetDefault(ctx context.Context, callerID, id int64) error {
	if err := s.repo.SetDefault(ctx, id, callerID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return errs.New(errs.CodeNotFound, "address not found")
		}
		return errs.Wrap(errs.CodeInternal, "set default", err)
	}
	return nil
}

// Delete 删除 caller 的地址。
func (s *Service) Delete(ctx context.Context, callerID, id int64) error {
	if err := s.repo.Delete(ctx, id, callerID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return errs.New(errs.CodeNotFound, "address not found")
		}
		return errs.Wrap(errs.CodeInternal, "delete address", err)
	}
	return nil
}

// ---------- helpers ----------

func validateFields(recipient, phone, detail string) error {
	if l := len(strings.TrimSpace(recipient)); l == 0 || l > 32 {
		return errs.New(errs.CodeParamInvalid, "recipient length must be 1..32")
	}
	if l := len(strings.TrimSpace(phone)); l < 7 || l > 20 {
		return errs.New(errs.CodeParamInvalid, "phone length must be 7..20")
	}
	if l := len(strings.TrimSpace(detail)); l == 0 || l > 200 {
		return errs.New(errs.CodeParamInvalid, "detail length must be 1..200")
	}
	return nil
}

// ---------- handler ----------

// Handler 把 address 业务暴露为 REST 端点。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把路由挂到 g（通常由 router 包传入 /api/v1 group）。
// 所有路由都已经在 router 层挂好 Auth 中间件，handler 不再校验 caller。
func (h *Handler) RegisterRoutes(g gin.IRouter) {
	addr := g.Group("/addresses")
	addr.GET("", h.list)
	addr.POST("", h.create)
	addr.PUT("/:id", h.update)
	addr.PUT("/:id/default", h.setDefault)
	addr.DELETE("/:id", h.delete)
}

type listResp struct {
	Items []gin.H `json:"items"`
}

func (h *Handler) list(c *gin.Context) {
	caller := callerID(c)
	list, err := h.svc.List(c.Request.Context(), caller)
	if err != nil {
		respondError(c, err)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, a := range list {
		items = append(items, toJSON(a))
	}
	httpx.OK(c, listResp{Items: items})
}

func (h *Handler) create(c *gin.Context) {
	caller := callerID(c)
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	a, err := h.svc.Create(c.Request.Context(), caller, in)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(a))
}

func (h *Handler) update(c *gin.Context) {
	caller := callerID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	a, err := h.svc.Update(c.Request.Context(), caller, id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(a))
}

func (h *Handler) setDefault(c *gin.Context) {
	caller := callerID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.SetDefault(c.Request.Context(), caller, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "is_default": true})
}

func (h *Handler) delete(c *gin.Context) {
	caller := callerID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), caller, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "deleted": true})
}

// ---------- helpers ----------

// callerID 从 gin ctx 读 user_id（中间件注入）。
func callerID(c *gin.Context) int64 {
	v, _ := c.Get("user_user_id")
	id, _ := v.(int64)
	return id
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
	if id == 0 {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return 0, false
	}
	return id, true
}

func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

func toJSON(a *Record) gin.H {
	return gin.H{
		"id":         a.ID,
		"user_id":    a.UserID,
		"recipient":  a.Recipient,
		"phone":      a.Phone,
		"detail":     a.Detail,
		"lat":        a.Lat,
		"lng":        a.Lng,
		"is_default": a.IsDefault,
		"created_at": a.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at": a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}