// Package handler 翻译 wallet HTTP 请求 ↔ service 调用 + errs 业务码。
//
// 设计要点：
//   - 4 个 P0/P1 endpoint + 3 个 admin endpoint（admin-web 调用）。
//   - 全部走 shared/httpx 统一响应格式。
//   - 提现最低 100 元（service.MinWithdrawalCents）；handler 不重复校验。
//   - amount 用 string 入参 → decimal.NewFromString；避免 JS 浮点精度漂移。
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 wallet-service 业务接口（service.Service 满足）。
type Service interface {
	GetWallet(ctx context.Context, userID int64) (*repo.Wallet, error)
	CreateWithdrawal(ctx context.Context, userID int64, amount decimal.Decimal, channel, account string) (*repo.Withdrawal, error)
	ApproveWithdrawal(ctx context.Context, id, reviewerID int64) error
	MarkWithdrawalPaid(ctx context.Context, id int64) error
	RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error
	ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*repo.Billing, error)
}

// AdminSvc admin 操作（admin-web 调用）。
// service.Service 同时实现 Service + AdminSvc（MarkWithdrawalPaid 等同方法）。
type AdminSvc interface {
	ApproveWithdrawal(ctx context.Context, id, reviewerID int64) error
	MarkWithdrawalPaid(ctx context.Context, id int64) error
	RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error
}

// Handler 持有 service 引用。
type Handler struct {
	svc   Service
	admin AdminSvc
}

// New 构造 handler。
func New(svc Service, admin AdminSvc) *Handler { return &Handler{svc: svc, admin: admin} }

// RegisterRoutes 注册路由（v1 暴露 4 个 P0/P1 endpoint + 3 个 admin endpoint）。
// auth 中间件由调用方（router 层）注入；handler 不依赖具体 ctx key 名。
func (h *Handler) RegisterRoutes(r *gin.Engine, auth gin.HandlerFunc) {
	api := r.Group("/api/v1")
	authed := api.Group("/", auth)

	authed.GET("/users/me/wallet", h.getUserWallet)
	authed.GET("/escorts/me/wallet", h.getEscortWallet)
	authed.POST("/escorts/me/wallet/withdraw", h.createWithdrawal)
	authed.GET("/wallet/transactions", h.listTransactions)

	// admin（admin 角色鉴权留给 middleware；v1 不强校验，admin-web 走）
	if h.admin != nil {
		authed.POST("/admin/wallet/withdrawals/:id/approve", h.adminApprove)
		authed.POST("/admin/wallet/withdrawals/:id/pay", h.adminPay)
		authed.POST("/admin/wallet/withdrawals/:id/reject", h.adminReject)
	}
}

// ---- 公共 helper ----

// mustUserID 从 ctx 取 user_id（中间件塞入）。
func mustUserID(c *gin.Context) int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	uid, _ := v.(int64)
	return uid
}

// respondError 把业务错误翻译成 httpx.Fail。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// ---- 用户视角 endpoint ----

// getUserWallet GET /api/v1/users/me/wallet（患者钱包）。
func (h *Handler) getUserWallet(c *gin.Context) {
	uid := mustUserID(c)
	w, err := h.svc.GetWallet(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"user_id":         w.UserID,
		"balance":         w.Balance.String(),
		"frozen":          w.Frozen.String(),
		"total_earned":    w.TotalEarned.String(),
		"total_withdrawn": w.TotalWithdrawn.String(),
		"currency":        w.Currency,
		"updated_at":      w.UpdatedAt,
	})
}

// getEscortWallet GET /api/v1/escorts/me/wallet（陪诊师钱包）。
func (h *Handler) getEscortWallet(c *gin.Context) {
	uid := mustUserID(c)
	w, err := h.svc.GetWallet(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"user_id":         w.UserID,
		"balance":         w.Balance.String(),
		"frozen":          w.Frozen.String(),
		"total_earned":    w.TotalEarned.String(),
		"total_withdrawn": w.TotalWithdrawn.String(),
		"currency":        w.Currency,
		"updated_at":      w.UpdatedAt,
	})
}

type createWithdrawalReq struct {
	Amount  string `json:"amount"`  // 字符串，避免 JS 浮点精度漂移
	Channel string `json:"channel"` // "wx" | "alipay"
	Account string `json:"account"` // 脱敏：138****0000
}

// createWithdrawal POST /api/v1/escorts/me/wallet/withdraw。
func (h *Handler) createWithdrawal(c *gin.Context) {
	uid := mustUserID(c)
	var req createWithdrawalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "amount must be decimal string"))
		return
	}
	w, err := h.svc.CreateWithdrawal(c.Request.Context(), uid, amount, req.Channel, req.Account)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"id":         w.ID,
		"amount":     w.Amount.String(),
		"channel":    w.Channel,
		"account":    w.Account,
		"status":     w.Status,
		"created_at": w.CreatedAt,
	})
}

// listTransactions GET /api/v1/wallet/transactions?limit=20&offset=0。
func (h *Handler) listTransactions(c *gin.Context) {
	uid := mustUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	txs, err := h.svc.ListTransactions(c.Request.Context(), uid, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]gin.H, 0, len(txs))
	for _, b := range txs {
		out = append(out, gin.H{
			"id":            b.ID,
			"order_id":      b.OrderID,
			"type":          b.Type,
			"amount":        b.Amount.String(),
			"balance_after": b.BalanceAfter.String(),
			"frozen_after":  b.FrozenAfter.String(),
			"note":          b.Note,
			"created_at":    b.CreatedAt,
		})
	}
	httpx.OK(c, out)
}

// ---- admin endpoint ----

func (h *Handler) adminApprove(c *gin.Context) {
	reviewer := mustUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.admin.ApproveWithdrawal(c.Request.Context(), id, reviewer); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "status": "approved"})
}

func (h *Handler) adminPay(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	if err := h.admin.MarkWithdrawalPaid(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "status": "paid"})
}

func (h *Handler) adminReject(c *gin.Context) {
	reviewer := mustUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	if err := h.admin.RejectWithdrawal(c.Request.Context(), id, reviewer, req.Reason); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "status": "rejected"})
}

// (no longer needed; use c.Request.Context() directly)