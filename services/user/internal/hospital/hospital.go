// Package hospital 实现 user-service 医院库的 service + handler。
//
// 设计要点：
//   - 仅 2 个 GET 端点（列表 + 详情）；admin CRUD 由 admin plan 接管。
//   - 分页契约：page/limit；page 默认 1，limit 默认 20、最大 100。
package hospital

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Repository 是 hospital 模块的仓储契约。
type Repository interface {
	List(ctx context.Context, f ListFilter) ([]*Record, int, error)
	GetByID(ctx context.Context, id int64) (*Record, error)
}

// Service 是 hospital 业务编排器。
type Service struct {
	repo Repository
}

// NewService 装配。
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// ListResult 是列表返回（含分页）。
type ListResult struct {
	Items []*Record `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

// List 列出医院。
func (s *Service) List(ctx context.Context, page, limit int, cityID int64, level, keyword string) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	items, total, err := s.repo.List(ctx, ListFilter{
		CityID:  cityID,
		Level:   strings.TrimSpace(level),
		Keyword: keyword,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list hospitals", err)
	}
	return &ListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

// Get 医院详情。
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	h, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "hospital not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get hospital", err)
	}
	return h, nil
}

// ---------- handler ----------

// Handler 把 hospital 业务暴露为 REST。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂路由。
func (h *Handler) RegisterRoutes(g gin.IRouter) {
	g.GET("/hospitals", h.list)
	g.GET("/hospitals/:id", h.get)
}

func (h *Handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cityID, _ := strconv.ParseInt(c.DefaultQuery("city_id", "0"), 10, 64)
	level := c.Query("level")
	keyword := c.Query("keyword")
	out, err := h.svc.List(c.Request.Context(), page, limit, cityID, level, keyword)
	if err != nil {
		respondError(c, err)
		return
	}
	items := make([]gin.H, 0, len(out.Items))
	for _, x := range out.Items {
		items = append(items, toJSON(x))
	}
	httpx.OK(c, gin.H{
		"items": items,
		"total": out.Total,
		"page":  out.Page,
		"limit": out.Limit,
	})
}

func (h *Handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	r, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toJSON(r))
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

func toJSON(h *Record) gin.H {
	return gin.H{
		"id":          h.ID,
		"name":        h.Name,
		"city_id":     h.CityID,
		"level":       h.Level,
		"status":      h.Status,
		"address":     h.Address,
		"lat":         h.Lat,
		"lng":         h.Lng,
		"phone":       h.Phone,
		"departments": h.Departments,
		"description": h.Description,
	}
}