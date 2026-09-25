package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/payment/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- fake ----------

// fakeSvc 实现 handler.Service 接口。
type fakeSvc struct {
	createOut   *service.Payment
	createErr   error
	getOut      *service.Payment
	getErr      error
	completeOut *service.Payment
	completeErr error
	refundErr   error
	refundCalls int
}

func (f *fakeSvc) Create(ctx context.Context, orderID int64, amount float64) (*service.Payment, error) {
	return f.createOut, f.createErr
}
func (f *fakeSvc) Get(ctx context.Context, paymentID int64) (*service.Payment, error) {
	return f.getOut, f.getErr
}
func (f *fakeSvc) Complete(ctx context.Context, paymentID int64, externalTxID string) (*service.Payment, error) {
	return f.completeOut, f.completeErr
}
func (f *fakeSvc) Refund(ctx context.Context, paymentID int64) error {
	f.refundCalls++
	return f.refundErr
}

// 编译期断言 fakeSvc 满足 Service 接口。
var _ Service = (*fakeSvc)(nil)

// fakeAuth 返回一个把 user_id 塞进 ctx 的中间件（绕开 JWT 校验）。
// 缺 user_id query 时返回 401，便于测试拒绝路径。
func fakeAuth(uid int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Query("user_id") != "" {
			c.Set("user_id", uid)
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

// mkRouter 构造带 fake auth 的 gin engine。
func mkRouter(svc Service, uid int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1", fakeAuth(uid))
	h := New(svc)
	h.RegisterRoutes(api)
	return r
}

func ptrTime(t time.Time) *time.Time { return &t }

// ---------- 业务码常量（避免 hard-code 散落） ----------

const (
	codeParamInvalid = int(errs.CodeParamInvalid) // 10001
	codeNotFound     = int(errs.CodeNotFound)     // 12001
	codeConflict     = int(errs.CodeConflict)     // 12002
)

// ---------- Create ----------

// TestCreate_OK 验证 POST /api/v1/payments 成功。
func TestCreate_OK(t *testing.T) {
	p := &service.Payment{ID: 1, OrderID: 100, Amount: 200, Channel: "mock", Status: "created"}
	svc := &fakeSvc{createOut: p}
	r := mkRouter(svc, 1)

	body, _ := json.Marshal(map[string]any{"order_id": 100, "amount": 200.0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.EqualValues(t, 1, resp.Data["id"])
	assert.EqualValues(t, 200, resp.Data["amount"])
	assert.Equal(t, "created", resp.Data["status"])
}

// TestCreate_BindError 验证缺字段（order_id / amount）→ 10001。
func TestCreate_BindError(t *testing.T) {
	svc := &fakeSvc{}
	r := mkRouter(svc, 1)
	body, _ := json.Marshal(map[string]any{}) // 缺 order_id / amount
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeParamInvalid, resp.Code)
}

// TestCreate_BadAmount 验证 amount <= 0 → 10001。
func TestCreate_BadAmount(t *testing.T) {
	svc := &fakeSvc{}
	r := mkRouter(svc, 1)
	body, _ := json.Marshal(map[string]any{"order_id": 100, "amount": 0.0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeParamInvalid, resp.Code)
}

// TestCreate_DuplicateOrder 验证 service 返回 CodeConflict → 12002 透传。
func TestCreate_DuplicateOrder(t *testing.T) {
	svc := &fakeSvc{createErr: errs.New(errs.CodeConflict, "payment already exists for order")}
	r := mkRouter(svc, 1)
	body, _ := json.Marshal(map[string]any{"order_id": 100, "amount": 200.0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeConflict, resp.Code)
}

// ---------- Complete ----------

// TestComplete_OK 验证 POST /api/v1/payments/:id/complete。
func TestComplete_OK(t *testing.T) {
	now := time.Now()
	p := &service.Payment{ID: 1, OrderID: 100, Amount: 200, Channel: "mock", Status: "completed", CompletedAt: ptrTime(now)}
	svc := &fakeSvc{completeOut: p}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/complete?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "completed", resp.Data["status"])
}

// TestComplete_NotFound 验证未找到 → 12001。
func TestComplete_NotFound(t *testing.T) {
	svc := &fakeSvc{completeErr: service.ErrPaymentNotFound}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/999/complete?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeNotFound, resp.Code)
}

// ---------- Refund ----------

// TestRefund_OK 验证 POST /api/v1/payments/:id/refund 成功。
func TestRefund_OK(t *testing.T) {
	svc := &fakeSvc{}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/refund?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.EqualValues(t, 1, resp.Data["id"])
	assert.Equal(t, "refunded", resp.Data["status"])
	assert.Equal(t, 1, svc.refundCalls)
}

// TestRefund_NotCompleted 验证未支付 → 12002。
func TestRefund_NotCompleted(t *testing.T) {
	svc := &fakeSvc{refundErr: errs.New(errs.CodeConflict, "only completed payments can be refunded")}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/refund?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeConflict, resp.Code)
}

// ---------- Get ----------

// TestGet_OK 验证 GET /api/v1/payments/:id。
func TestGet_OK(t *testing.T) {
	p := &service.Payment{ID: 7, OrderID: 100, Amount: 200, Channel: "mock", Status: "completed"}
	svc := &fakeSvc{getOut: p}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/7?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.EqualValues(t, 7, resp.Data["id"])
}

// TestGet_NotFound 验证未找到 → 12001。
func TestGet_NotFound(t *testing.T) {
	svc := &fakeSvc{getErr: service.ErrPaymentNotFound}
	r := mkRouter(svc, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/999?user_id=1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, codeNotFound, resp.Code)
}

// 引用避免 unused warning（context 包间接使用）
var _ = context.Background