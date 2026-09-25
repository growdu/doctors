// Package virtualnumber 实现 user-service 虚拟号的 service + handler。
//
// 设计要点：
//   - 2 个端点：POST /virtual-numbers/allocate（创建）、GET /virtual-numbers/:id（详情）；
//   - v1 不接 Kafka 广播（注释说明 v2 接 message/notification 时再发布 contracts.VirtualNumberAllocatedEvent）；
//   - status 重算在 repo 读取时进行。
package virtualnumber

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Repository 是 virtualnumber 模块的仓储契约。
type Repository interface {
	Allocate(ctx context.Context, in AllocateInput) (*Record, error)
	GetByID(ctx context.Context, id int64) (*Record, error)
	Release(ctx context.Context, id int64, reason string) (*Record, error)
}

// Service 是 virtualnumber 业务编排器。
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService 装配。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now() }}
}

// AllocateRequest 是 service Allocate 入参（不含 phone，v1 自动生成）。
type AllocateRequest struct {
	OrderID   int64     `json:"order_id"`
	PatientID int64     `json:"patient_id"`
	EscortID  int64     `json:"escort_id"`
	ExpireAt  time.Time `json:"expire_at"`
}

// Allocate 分配虚拟号；参数校验 + 唯一性由 DB partial unique 保证。
func (s *Service) Allocate(ctx context.Context, in AllocateRequest) (*Record, error) {
	if in.OrderID <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "order_id must be > 0")
	}
	if in.PatientID <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "patient_id must be > 0")
	}
	if in.EscortID <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "escort_id must be > 0")
	}
	if in.ExpireAt.IsZero() || !in.ExpireAt.After(s.now()) {
		return nil, errs.New(errs.CodeParamInvalid, "expire_at must be in the future")
	}
	v, err := s.repo.Allocate(ctx, AllocateInput{
		OrderID:   in.OrderID,
		PatientID: in.PatientID,
		EscortID:  in.EscortID,
		ExpireAt:  in.ExpireAt,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateActive) {
			return nil, errs.New(errs.CodeConflict, "order already has active virtual number")
		}
		return nil, errs.Wrap(errs.CodeInternal, "allocate virtual number", err)
	}
	return v, nil
}

// Get 详情。
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "virtual number not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get virtual number", err)
	}
	return v, nil
}

// Release 释放虚拟号（v1 暂不接管理端，仅供 service 内部使用；v2 接 admin 后再加 handler）。
func (s *Service) Release(ctx context.Context, id int64, reason string) error {
	if _, err := s.repo.Release(ctx, id, reason); err != nil {
		if errors.Is(err, ErrNotFound) {
			return errs.New(errs.CodeNotFound, "virtual number not found")
		}
		return errs.Wrap(errs.CodeInternal, "release virtual number", err)
	}
	return nil
}

// ---------- handler ----------

// Handler 把 virtualnumber 业务暴露为 REST。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂路由。
func (h *Handler) RegisterRoutes(g gin.IRouter) {
	g.POST("/virtual-numbers/allocate", h.allocate)
	g.GET("/virtual-numbers/:id", h.get)
}

type allocateReq struct {
	OrderID   int64     `json:"order_id" binding:"required"`
	PatientID int64     `json:"patient_id" binding:"required"`
	EscortID  int64     `json:"escort_id" binding:"required"`
	ExpireAt  time.Time `json:"expire_at" binding:"required"`
}

func (h *Handler) allocate(c *gin.Context) {
	var req allocateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	v, err := h.svc.Allocate(c.Request.Context(), AllocateRequest{
		OrderID:   req.OrderID,
		PatientID: req.PatientID,
		EscortID:  req.EscortID,
		ExpireAt:  req.ExpireAt,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(v))
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(v))
}

// ---------- helpers ----------

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

func toJSON(v *Record) gin.H {
	return gin.H{
		"id":          v.ID,
		"order_id":    v.OrderID,
		"patient_id":  v.PatientID,
		"escort_id":   v.EscortID,
		"phone":       v.Phone,
		"status":      v.Status,
		"expire_at":   v.ExpireAt.UTC().Format(time.RFC3339),
		"released_at": formatTimePtr(v.ReleasedAt),
		"created_at":  v.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}