package router

import (
	"bytes"
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

	"github.com/growdu/doctors/services/payment/internal/handler"
	"github.com/growdu/doctors/services/payment/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "payment-router-secret"

// fakeRouterSvc 实现 handler.Service 接口（最小化；返回占位对象）。
type fakeRouterSvc struct{}

func (fakeRouterSvc) Create(ctx context.Context, orderID int64, amount float64) (*service.Payment, error) {
	return &service.Payment{
		ID: 1, OrderID: orderID, Amount: amount, Channel: "mock", Status: "created",
		CreatedAt: time.Now(),
	}, nil
}
func (fakeRouterSvc) Get(ctx context.Context, paymentID int64) (*service.Payment, error) {
	return &service.Payment{
		ID: paymentID, OrderID: 100, Amount: 200, Channel: "mock", Status: "created",
	}, nil
}
func (fakeRouterSvc) Complete(ctx context.Context, paymentID int64, externalTxID string) (*service.Payment, error) {
	now := time.Now()
	return &service.Payment{
		ID: paymentID, OrderID: 100, Amount: 200, Channel: "mock", Status: "completed",
		CompletedAt: &now,
	}, nil
}
func (fakeRouterSvc) Refund(ctx context.Context, paymentID int64) error { return nil }

var _ handler.Service = fakeRouterSvc{}

// signToken 签发一个测试用 JWT（user_id + role）。
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

// TestHealthz_NoAuth 验证 /healthz 不挂 auth → 200。
func TestHealthz_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(fakeRouterSvc{})
	r := New(h, testSecret)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Data["status"])
}

// TestCreate_NoToken_401 验证 POST /api/v1/payments 无 token → 11001。
func TestCreate_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(fakeRouterSvc{})
	r := New(h, testSecret)

	body, _ := json.Marshal(map[string]any{"order_id": 100, "amount": 200.0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int(11001), env.Code)
}

// TestCreate_WithToken_OK 验证带 token → 0 + 业务成功。
func TestCreate_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(fakeRouterSvc{})
	r := New(h, testSecret)
	tok := signToken(t, 1, "patient")

	body, _ := json.Marshal(map[string]any{"order_id": 100, "amount": 200.0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 0, env.Code)
}

// TestComplete_NoToken_401 验证 POST /api/v1/payments/:id/complete 无 token → 11001。
func TestComplete_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.New(fakeRouterSvc{})
	r := New(h, testSecret)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/1/complete", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var env httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int(11001), env.Code)
}