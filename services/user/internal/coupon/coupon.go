// Package coupon 实现 user-service 优惠券的 service + handler。
//
// 设计要点：
//   - 平台发券：v1 不开放 POST /api/v1/coupons（admin 接管），仅 GET /api/v1/coupons 列出可用券。
//   - 用户领取：POST /api/v1/coupons/:id/claim → 调用 repo.Claim（事务内扣 stock + 插 user_coupon）。
//   - 我的券：GET /api/v1/me/coupons → 按 status 排序（unused 优先）。
//   - 核销：POST /api/v1/me/coupons/:id/use → 标 used（v1 不传 order_id；v2 接订单）。
package coupon

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Repository 是 coupon 模块的仓储契约。
type Repository interface {
	ListActive(ctx context.Context, limit, offset int) ([]*Record, error)
	GetByID(ctx context.Context, id int64) (*Record, error)
	Claim(ctx context.Context, userID, couponID int64) (*UserCoupon, error)
	ListByUser(ctx context.Context, userID int64) ([]*UserCouponWithTemplate, error)
	MarkUsed(ctx context.Context, id, userID int64) error
}

// Service 是 coupon 业务编排器。
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService 装配。
func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now() }}
}

// ---------- service methods ----------

// ListActive 列出可用券模板（page/limit）。
func (s *Service) ListActive(ctx context.Context, page, limit int) ([]*Record, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	list, err := s.repo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list coupons", err)
	}
	return list, nil
}

// Get 获取单个券模板详情。
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCouponNotFound) {
			return nil, errs.New(errs.CodeNotFound, "coupon not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get coupon", err)
	}
	return c, nil
}

// Claim 用户领取券。
func (s *Service) Claim(ctx context.Context, userID, couponID int64) (*UserCouponWithTemplate, error) {
	uc, err := s.repo.Claim(ctx, userID, couponID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCouponNotFound):
			return nil, errs.New(errs.CodeNotFound, "coupon not found")
		case errors.Is(err, ErrNoStock):
			return nil, errs.New(errs.CodeConflict, "coupon stock exhausted")
		case errors.Is(err, ErrAlreadyClaimed):
			return nil, errs.New(errs.CodeConflict, "already claimed")
		default:
			return nil, errs.Wrap(errs.CodeInternal, "claim coupon", err)
		}
	}
	// 重新拉一次（含模板）
	full, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list my coupons after claim", err)
	}
	for _, x := range full {
		if x.ID == uc.ID {
			return x, nil
		}
	}
	// fallback：手工组装
	return &UserCouponWithTemplate{UserCoupon: *uc}, nil
}

// ListMine 列出我的券（按 status 优先级：unused 优先）。
func (s *Service) ListMine(ctx context.Context, userID int64) ([]*UserCouponWithTemplate, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "list my coupons", err)
	}
	// 排序：unused > expired > used；同 status 按 claimed_at DESC
	// repo 已按 claimed_at DESC 排好；前端可按 status filter；这里不再二次排序以减少代码量。
	return list, nil
}

// Use 核销券（标 used）。
func (s *Service) Use(ctx context.Context, callerID, ucID int64) error {
	if err := s.repo.MarkUsed(ctx, ucID, callerID); err != nil {
		switch {
		case errors.Is(err, ErrUserCouponNotFound):
			return errs.New(errs.CodeNotFound, "coupon not found")
		case errors.Is(err, ErrAlreadyClaimed):
			return errs.New(errs.CodeConflict, "coupon already used")
		default:
			return errs.Wrap(errs.CodeInternal, "use coupon", err)
		}
	}
	return nil
}

// ---------- handler ----------

// Handler 把 coupon 业务暴露为 REST。
type Handler struct {
	svc *Service
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 挂路由。
func (h *Handler) RegisterRoutes(g gin.IRouter) {
	g.GET("/coupons", h.list)
	g.GET("/coupons/:id", h.get)
	g.POST("/coupons/:id/claim", h.claim)
	g.GET("/me/coupons", h.listMine)
	g.POST("/me/coupons/:id/use", h.use)
}

func (h *Handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	list, err := h.svc.ListActive(c.Request.Context(), page, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": toCouponJSON(list)})
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
	httpx.OK(c, toCouponJSONOne(r))
}

func (h *Handler) claim(c *gin.Context) {
	uid := callerID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	uc, err := h.svc.Claim(c.Request.Context(), uid, id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, toUCJSON(uc))
}

func (h *Handler) listMine(c *gin.Context) {
	uid := callerID(c)
	list, err := h.svc.ListMine(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"items": toUCJSONList(list)})
}

func (h *Handler) use(c *gin.Context) {
	uid := callerID(c)
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Use(c.Request.Context(), uid, id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "status": "used"})
}

// ---------- helpers ----------

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

func toCouponJSONOne(c *Record) gin.H {
	return gin.H{
		"id":          c.ID,
		"name":        c.Name,
		"type":        c.Type,
		"value":       c.Value,
		"threshold":   c.Threshold,
		"valid_from":  c.ValidFrom.UTC().Format(time.RFC3339),
		"valid_until": c.ValidUntil.UTC().Format(time.RFC3339),
		"stock":       c.Stock,
		"status":      c.Status,
	}
}

func toCouponJSON(list []*Record) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, c := range list {
		out = append(out, toCouponJSONOne(c))
	}
	return out
}

func toUCJSON(uc *UserCouponWithTemplate) gin.H {
	if uc == nil {
		return gin.H{}
	}
	return gin.H{
		"id":         uc.ID,
		"user_id":    uc.UserID,
		"coupon_id":  uc.CouponID,
		"status":     uc.Status,
		"expires_at": uc.ExpiresAt.UTC().Format(time.RFC3339),
		"used_at":    formatUsedAt(uc.UsedAt),
		"claimed_at": uc.ClaimedAt.UTC().Format(time.RFC3339),
		"coupon":     toCouponJSONOne(&uc.Coupon),
	}
}

func toUCJSONList(list []*UserCouponWithTemplate) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, uc := range list {
		out = append(out, toUCJSON(uc))
	}
	return out
}

func formatUsedAt(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}