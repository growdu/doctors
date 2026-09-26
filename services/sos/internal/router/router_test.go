package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/sos/internal/handler"
	"github.com/growdu/doctors/services/sos/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "sos-router-secret"

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct{}

func (stubSvc) Raise(ctx context.Context, orderID, userID int64, lat, lng float64, note string) (*service.SOS, error) {
	return &service.SOS{ID: 1, OrderID: orderID, UserID: userID, Status: "raised"}, nil
}
func (stubSvc) GetByID(ctx context.Context, id int64) (*service.SOS, error) {
	return &service.SOS{ID: id, Status: "raised"}, nil
}
func (stubSvc) List(ctx context.Context, f service.ListFilter) ([]*service.SOS, error) {
	return nil, nil
}
func (stubSvc) Resolve(ctx context.Context, sosID int64) error { return nil }

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

// TestHealthz_NoAuth 验证 /healthz 不挂 auth。
func TestHealthz_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, 200, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestSOS_NoToken_401 验证 /api/v1/sos 无 token → 11001。
func TestSOS_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/sos", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestSOS_InvalidToken_401 验证非法 token → 11001。
func TestSOS_InvalidToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sos", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestRaise_WithToken_OK 验证 POST /api/v1/sos 带 token 走通。
func TestRaise_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "patient")
	body := `{"order_id":100,"lat":39.9,"lng":116.4}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sos", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestResolve_WithToken_OK 验证 POST /api/v1/sos/:id/resolve。
func TestResolve_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "super_admin")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sos/7/resolve", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestDetail_WithToken_OK 验证 GET /api/v1/sos/:id。
func TestDetail_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "patient")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sos/7", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}
