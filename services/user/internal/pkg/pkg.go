// Package pkg 实现 user-service 服务包的 service + handler。
//
// 设计要点：
//   - 2 个 GET 端点：list by hospital + detail。
//   - service 校验 hospital_id > 0；价格转 string 避免 JS 浮点漂移。
package pkg

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Repository 是 pkg 模块的仓储契约。
type Repository interface {
	ListByHospital(ctx context.Context, hospitalID int64) ([]*Record, error)
	GetByID(ctx context.Context, id int64) (*Record, error)
}

// Service 是 pkg 业务编排器。
type Service struct {
	repo Repository
}

// NewService 装配。
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListByHospital 列出医院的服务包。
func (s *Service) ListByHospital(ctx context.Context, hospitalID int64) ([]*Record, error) {
	if hospitalID <= 0 {
		return nil, errs.New(errs.CodeParamInvalid, "hospital_id must be > 0")
	}
	list, err := s.repo.ListByHospital(ctx, hospitalID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list packages", err)
	}
	return list, nil
}

// Get 服务包详情。
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "package not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get package", err)
	}
	return p, nil
}

// ---------- handler ----------

// Handler 把 pkg 业务暴露为 REST。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂路由。
//
// 注意：路由挂在 /hospitals/:id/packages 下 + 顶层 /packages/:id 详情。
// 这里通过 RegisterRoutes 把全部路径注册到给定的 group 上，
// 由调用方（router）确保 URL 路径已挂载在合适前缀。
func (h *Handler) RegisterRoutes(g gin.IRouter) {
	// 列表（按医院）
	g.GET("/hospitals/:id/packages", h.listByHospital)
	// 详情
	g.GET("/packages/:id", h.get)
}

func (h *Handler) listByHospital(c *gin.Context) {
	hid, ok := parseID(c)
	if !ok {
		return
	}
	list, err := h.svc.ListByHospital(c.Request.Context(), hid)
	if err != nil {
		respondError(c, err)
		return
	}
	items := make([]gin.H, 0, len(list))
	for _, p := range list {
		items = append(items, toJSON(p))
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(p))
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

func toJSON(p *Record) gin.H {
	// price 转为 string 避免 JS 浮点精度漂移（与 wallet 一致风格）
	return gin.H{
		"id":           p.ID,
		"hospital_id":  p.HospitalID,
		"name":         p.Name,
		"type":         p.Type,
		"duration_min": p.DurationMin,
		"price":        strconv.FormatFloat(p.Price, 'f', 2, 64),
		"status":       p.Status,
		"description":  p.Description,
	}
}