// Package handler 把 payment-service 业务能力翻译成 HTTP 接口。
//
// 设计要点：
//   - 4 个 P0 endpoint：POST /payments（按订单创建）/ GET /payments/:id（详情）
//     / POST /payments/:id/complete（mock 微信支付完成）/ POST /payments/:id/refund（申请退款）。
//   - 全部走 shared/httpx 统一响应格式（业务码在 body.code 中，HTTP 200）。
//   - 业务错误统一翻译：service.ErrPaymentNotFound → 12001；errs.Error 透传；其他 → 500000。
//   - 鉴权由 router 层注入（JWT 中间件），handler 只读 ctx 里的 user_id。
package handler

import (
	"context"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/payment/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// Service 是 handler 依赖的 service 接口（service.Service 满足）。
type Service interface {
	Create(ctx context.Context, orderID int64, amount float64) (*service.Payment, error)
	Get(ctx context.Context, paymentID int64) (*service.Payment, error)
	Complete(ctx context.Context, paymentID int64, externalTxID string) (*service.Payment, error)
	Refund(ctx context.Context, paymentID int64) error
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造 handler。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 注册路由（v1 暴露 4 个 endpoint）。
//
// auth 中间件由调用方（router 层）注入；handler 不依赖具体 ctx key 名（用通用 "user_id"）。
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/payments")
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
	g.POST("/:id/complete", h.Complete)
	g.POST("/:id/refund", h.Refund)
}

// respondError 统一错误翻译。
func respondError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrPaymentNotFound) {
		httpx.Fail(c, int(errs.CodeNotFound), "payment not found")
		return
	}
	if e, ok := errs.As(err); ok {
		httpx.Fail(c, int(e.Code), e.Msg)
		return
	}
	httpx.Fail(c, int(errs.CodeInternal), err.Error())
}

// mustUserID 从 ctx 取 user_id（中间件塞入；非 0 表示已登录）。
func mustUserID(c *gin.Context) int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	uid, _ := v.(int64)
	return uid
}

// parseID 把 :id 解析为 int64。
func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, "invalid id"))
		return 0, false
	}
	return id, true
}

// renderPayment 把 Payment 渲染成 map[string]any（统一响应）。
func renderPayment(p *service.Payment) gin.H {
	return gin.H{
		"id":             p.ID,
		"order_id":       p.OrderID,
		"amount":         p.Amount,
		"channel":        p.Channel,
		"external_tx_id": p.ExternalTxID,
		"status":         p.Status,
		"created_at":     p.CreatedAt,
		"completed_at":   p.CompletedAt,
		"refunded_at":    p.RefundedAt,
	}
}

// ---------- endpoints ----------

type createReq struct {
	OrderID int64   `json:"order_id" binding:"required"`
	Amount  float64 `json:"amount" binding:"required"`
}

// Create POST /api/v1/payments（按订单创建支付单）。
func (h *Handler) Create(c *gin.Context) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.Wrap(errs.CodeParamInvalid, "bind json", err))
		return
	}
	if req.Amount <= 0 {
		respondError(c, errs.New(errs.CodeParamInvalid, "amount must be > 0"))
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, renderPayment(p))
}

// Get GET /api/v1/payments/:id（支付单详情）。
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, renderPayment(p))
}

type completeReq struct {
	ExternalTxID string `json:"external_tx_id"`
}

// Complete POST /api/v1/payments/:id/complete（mock 微信支付完成回调）。
//
// body 可选；external_tx_id 不传时使用 service.Channel.CreateOutTradeNo 生成的 mock 流水号。
func (h *Handler) Complete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req completeReq
	_ = c.ShouldBindJSON(&req)
	p, err := h.svc.Complete(c.Request.Context(), id, req.ExternalTxID)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, renderPayment(p))
}

// Refund POST /api/v1/payments/:id/refund（申请退款）。
//
// 仅 completed 状态的支付单可退款；其他状态 service 透传 errs.CodeConflict (12002)。
func (h *Handler) Refund(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Refund(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK[any](c, gin.H{"id": id, "status": "refunded"})
}