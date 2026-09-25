package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/admin/internal/handler"
	"github.com/growdu/doctors/services/admin/internal/repo"
	"github.com/growdu/doctors/services/admin/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "admin-router-secret"

// stubAdminSvc 实现 handler.Service 接口（最小化；返回 nil / 空）。
type stubAdminSvc struct{}

func (stubAdminSvc) ListUsers(ctx context.Context, _, _ string, _, _ int) ([]map[string]any, error) {
	return nil, nil
}
func (stubAdminSvc) ListOrders(ctx context.Context, _ service.ListOrdersParams) ([]map[string]any, error) {
	return nil, nil
}
func (stubAdminSvc) ForceCancelOrder(ctx context.Context, _, _ int64, _ string) error {
	return nil
}
func (stubAdminSvc) ListPendingEscorts(ctx context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (stubAdminSvc) ApproveEscort(ctx context.Context, _, _ int64, _ string) error { return nil }
func (stubAdminSvc) RejectEscort(ctx context.Context, _, _ int64, _ string) error  { return nil }
func (stubAdminSvc) ListRefunds(ctx context.Context, _ string, _, _ int) ([]map[string]any, error) {
	return nil, nil
}
func (stubAdminSvc) ApproveRefund(ctx context.Context, _, _, _ int64, _ string) error { return nil }
func (stubAdminSvc) RejectRefund(ctx context.Context, _, _, _ int64, _ string) error  { return nil }
func (stubAdminSvc) ListWorkOrders(ctx context.Context, _ repo.WorkOrderListFilter) ([]*repo.WorkOrder, error) {
	return nil, nil
}
func (stubAdminSvc) CreateWorkOrder(ctx context.Context, _ *repo.WorkOrder) error { return nil }
func (stubAdminSvc) AssignWorkOrder(ctx context.Context, _, _ int64) error         { return nil }
func (stubAdminSvc) ResolveWorkOrder(ctx context.Context, _ int64, _ string) error { return nil }
func (stubAdminSvc) OverviewStats(ctx context.Context) (*service.OverviewStats, error) {
	return &service.OverviewStats{}, nil
}

func signToken(t *testing.T, uid int64, role string) string {
	t.Helper()
	now := time.Now()
	tok, err := authpkg.Sign(testSecret, authpkg.Claims{
		UserID: uid, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})
	require.NoError(t, err)
	return tok
}

// TestHealthz 验证 /healthz 不挂 auth。
func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, 200, w.Code, w.Body.String())

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Data["status"])
}

// TestUsers_NoToken_401 验证无 token → 401。
func TestUsers_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil))
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotEqual(t, 0, env.Code)
	assert.Equal(t, 11001, env.Code, "无 token → 401/11001")
}

// TestUsers_Viewer_OK 验证 viewer 命中白名单（读权限）。
func TestUsers_Viewer_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "viewer")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code)
}

// TestForceCancel_Viewer_Forbidden 验证 viewer 调 force-cancel → 403 (11003)。
func TestForceCancel_Viewer_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "viewer")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orders/7/force-cancel", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 11003, env.Code)
}

// TestForceCancel_OrderAdmin_OK 验证 order_admin 调 force-cancel 命中。
func TestForceCancel_OrderAdmin_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "order_admin")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orders/7/force-cancel", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code, env.Message)
}

// TestEscorts_AuditAdmin_OK 验证 audit_admin 通过陪诊师审核。
func TestEscorts_AuditAdmin_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "audit_admin")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/escorts/5/approve", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code)
}

// TestRefunds_CS_Forbidden 验证 cs 不能审批退款（应 403）。
func TestRefunds_CS_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "cs")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/refunds/3/approve", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 11003, env.Code)
}

// TestRefunds_RefundAdmin_OK 验证 refund_admin 通过审批。
func TestRefunds_RefundAdmin_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "refund_admin")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/refunds/3/approve", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code)
}

// TestWorkOrders_ViewerForbiddenCreate 验证 viewer 不能创建工单（应 403）。
func TestWorkOrders_ViewerForbiddenCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(stubAdminSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "viewer")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/work-orders", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 11003, env.Code)
}