// Package handler 翻译 admin HTTP 请求 ↔ service 调用 + errs 业务码。
//
// 设计要点：
//   - admin 12 API 全部要求 JWT（Auth + RoleAuth 双层中间件，在 router 注册）。
//   - 强制取消 / 审核等写操作只接 super_admin / *_admin 角色；查询接全 6 角色。
//   - errors.As 优先翻译为 errs.Error，其它走 CodeInternal。
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/services/admin/internal/repo"
	"github.com/growdu/doctors/services/admin/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
	"github.com/growdu/doctors/shared/middleware"
)

// Service 是 handler 依赖的 service 接口。
type Service interface {
	ListUsers(ctx context.Context, role, keyword string, page, pageSize int) ([]map[string]any, error)
	ListOrders(ctx context.Context, p service.ListOrdersParams) ([]map[string]any, error)
	ForceCancelOrder(ctx context.Context, orderID, adminID int64, reason string) error

	ListPendingEscorts(ctx context.Context) ([]map[string]any, error)
	ApproveEscort(ctx context.Context, escortID, adminID int64, note string) error
	RejectEscort(ctx context.Context, escortID, adminID int64, note string) error

	ListRefunds(ctx context.Context, status string, page, pageSize int) ([]map[string]any, error)
	ApproveRefund(ctx context.Context, refundID, orderID, adminID int64, note string) error
	RejectRefund(ctx context.Context, refundID, orderID, adminID int64, note string) error

	ListWorkOrders(ctx context.Context, f repo.WorkOrderListFilter) ([]*repo.WorkOrder, error)
	CreateWorkOrder(ctx context.Context, w *repo.WorkOrder) error
	AssignWorkOrder(ctx context.Context, id, adminID int64) error
	ResolveWorkOrder(ctx context.Context, id int64, resolution string) error

	OverviewStats(ctx context.Context) (*service.OverviewStats, error)
}

// Handler 持有 service 引用。
type Handler struct {
	svc Service
}

// New 构造。
func New(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes 把 admin 12 API 挂到 v1 group。
//
// Auth 中间件已先挂在 v1 上；本函数按 API 区分 RoleAuth 白名单。
func (h *Handler) RegisterRoutes(v1 gin.IRouter) {
	admin := v1.Group("/admin")

	// 全 6 角色可读
	readRoles := []string{"super_admin", "order_admin", "refund_admin", "audit_admin", "cs", "viewer"}

	// Users：全 6 角色
	users := admin.Group("/users")
	users.Use(middleware.RoleAuth(readRoles...))
	users.GET("", h.ListUsers)

	// Orders：读 = super_admin / order_admin / refund_admin / cs / viewer（除 audit_admin）；
	//        写 = super_admin / order_admin
	orders := admin.Group("/orders")
	orders.GET("", middleware.RoleAuth(
		"super_admin", "order_admin", "refund_admin", "cs", "viewer"), h.ListOrders)
	orders.POST("/:id/force-cancel",
		middleware.RoleAuth("super_admin", "order_admin"),
		h.ForceCancelOrder)

	// Escorts：读 = 全 6 角色；写 = super_admin / audit_admin
	escorts := admin.Group("/escorts")
	escorts.GET("/pending-audit", middleware.RoleAuth(readRoles...), h.ListPendingEscorts)
	escorts.POST("/:id/approve",
		middleware.RoleAuth("super_admin", "audit_admin"),
		h.ApproveEscort)
	escorts.POST("/:id/reject",
		middleware.RoleAuth("super_admin", "audit_admin"),
		h.RejectEscort)

	// Refunds：读 = super_admin / refund_admin / cs / viewer；写 = super_admin / refund_admin
	refunds := admin.Group("/refunds")
	refunds.GET("", middleware.RoleAuth(
		"super_admin", "refund_admin", "cs", "viewer"), h.ListRefunds)
	refunds.POST("/:id/approve",
		middleware.RoleAuth("super_admin", "refund_admin"),
		h.ApproveRefund)
	refunds.POST("/:id/reject",
		middleware.RoleAuth("super_admin", "refund_admin"),
		h.RejectRefund)

	// Work Orders：全 6 角色可读；写 = 除 viewer 外 5 角色
	wo := admin.Group("/work-orders")
	wo.GET("", middleware.RoleAuth(readRoles...), h.ListWorkOrders)
	wo.POST("",
		middleware.RoleAuth("super_admin", "order_admin", "refund_admin", "audit_admin", "cs"),
		h.CreateWorkOrder)

	// Billings：全 6 角色（v1 简化：返回今日 GMV + 订单数）
	billings := admin.Group("/billings")
	billings.GET("", middleware.RoleAuth(readRoles...), h.ListBillings)

	// Reports overview：全 6 角色
	reports := admin.Group("/reports")
	reports.GET("/overview", middleware.RoleAuth(readRoles...), h.Overview)
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

// adminIDFromCtx 提取 admin id（JWT 已注入）。
func adminIDFromCtx(c *gin.Context) int64 {
	if v, ok := c.Get("uid"); ok {
		if uid, ok := v.(int64); ok {
			return uid
		}
	}
	return 0
}

// ---------- Users ----------

// ListUsers GET /api/v1/admin/users
func (h *Handler) ListUsers(c *gin.Context) {
	role := c.Query("role")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	out, err := h.svc.ListUsers(c.Request.Context(), role, keyword, page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"users": out, "page": page, "page_size": pageSize})
}

// ---------- Orders ----------

// ListOrders GET /api/v1/admin/orders
func (h *Handler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	hospitalID, _ := strconv.ParseInt(c.Query("hospital_id"), 10, 64)
	patientID, _ := strconv.ParseInt(c.Query("patient_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	out, err := h.svc.ListOrders(c.Request.Context(), service.ListOrdersParams{
		Status: status, HospitalID: hospitalID, PatientID: patientID,
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"orders": out})
}

// ForceCancelOrder POST /api/v1/admin/orders/:id/force-cancel
func (h *Handler) ForceCancelOrder(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	adminID := adminIDFromCtx(c)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ForceCancelOrder(c.Request.Context(), id, adminID, req.Reason); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"order_id": id, "status": "force_cancelled"})
}

// ---------- Escorts ----------

// ListPendingEscorts GET /api/v1/admin/escorts/pending-audit
func (h *Handler) ListPendingEscorts(c *gin.Context) {
	out, err := h.svc.ListPendingEscorts(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"escorts": out})
}

// ApproveEscort POST /api/v1/admin/escorts/:id/approve
func (h *Handler) ApproveEscort(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	adminID := adminIDFromCtx(c)
	var req struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ApproveEscort(c.Request.Context(), id, adminID, req.Note); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"escort_id": id, "status": "approved"})
}

// RejectEscort POST /api/v1/admin/escorts/:id/reject
func (h *Handler) RejectEscort(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	adminID := adminIDFromCtx(c)
	var req struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.RejectEscort(c.Request.Context(), id, adminID, req.Note); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"escort_id": id, "status": "rejected"})
}

// ---------- Refunds ----------

// ListRefunds GET /api/v1/admin/refunds
func (h *Handler) ListRefunds(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListRefunds(c.Request.Context(), status, page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"refunds": out})
}

// ApproveRefund POST /api/v1/admin/refunds/:id/approve
func (h *Handler) ApproveRefund(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	adminID := adminIDFromCtx(c)
	var req struct {
		OrderID int64  `json:"order_id"`
		Note    string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.ApproveRefund(c.Request.Context(), id, req.OrderID, adminID, req.Note); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"refund_id": id, "status": "approved"})
}

// RejectRefund POST /api/v1/admin/refunds/:id/reject
func (h *Handler) RejectRefund(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	adminID := adminIDFromCtx(c)
	var req struct {
		OrderID int64  `json:"order_id"`
		Note    string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.RejectRefund(c.Request.Context(), id, req.OrderID, adminID, req.Note); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"refund_id": id, "status": "rejected"})
}

// ---------- Work Orders ----------

// ListWorkOrders GET /api/v1/admin/work-orders
func (h *Handler) ListWorkOrders(c *gin.Context) {
	f := repo.WorkOrderListFilter{
		Status:   c.Query("status"),
		Category: c.Query("category"),
		Page:     1,
		PageSize: 20,
	}
	if v := c.Query("page"); v != "" {
		f.Page, _ = strconv.Atoi(v)
	}
	if v := c.Query("page_size"); v != "" {
		f.PageSize, _ = strconv.Atoi(v)
	}
	if v := c.Query("assignee_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AssigneeID = &id
		}
	}
	if v := c.Query("subject_type"); v != "" {
		f.SubjectType = v
	}
	out, err := h.svc.ListWorkOrders(c.Request.Context(), f)
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"work_orders": out})
}

// CreateWorkOrder POST /api/v1/admin/work-orders
func (h *Handler) CreateWorkOrder(c *gin.Context) {
	var req struct {
		UserID      int64   `json:"user_id" binding:"required"`
		Category    string  `json:"category" binding:"required"`
		Priority    string  `json:"priority"`
		SubjectID   *int64  `json:"subject_id"`
		SubjectType *string `json:"subject_type"`
		Title       string  `json:"title" binding:"required"`
		Content     string  `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, errs.New(errs.CodeParamInvalid, err.Error()))
		return
	}
	if req.Priority == "" {
		req.Priority = "P2"
	}
	w := &repo.WorkOrder{
		UserID:      req.UserID,
		Category:    req.Category,
		Priority:    req.Priority,
		Status:      "pending",
		SubjectID:   req.SubjectID,
		SubjectType: req.SubjectType,
		Title:       req.Title,
		Content:     req.Content,
	}
	if err := h.svc.CreateWorkOrder(c.Request.Context(), w); err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": w.ID, "status": w.Status})
}

// ---------- Billings（v1 stub：从 ReportsRepo Overview 简化） ----------

// ListBillings GET /api/v1/admin/billings（v1 简化版：返回今日 GMV + 订单数）。
func (h *Handler) ListBillings(c *gin.Context) {
	stats, err := h.svc.OverviewStats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, gin.H{
		"today_gmv":    stats.TodayGMV,
		"today_orders": stats.TodayOrders,
	})
}

// ---------- Reports ----------

// Overview GET /api/v1/admin/reports/overview
func (h *Handler) Overview(c *gin.Context) {
	stats, err := h.svc.OverviewStats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	httpx.OK(c, stats)
}