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

	"github.com/growdu/doctors/services/message/internal/handler"
	"github.com/growdu/doctors/services/message/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "message-router-secret"

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct{}

func (stubSvc) SendMessage(ctx context.Context, orderID, fromID, toID int64, body string) (*service.Message, error) {
	return &service.Message{ID: 1, OrderID: orderID, FromUserID: fromID, ToUserID: toID, Body: body}, nil
}
func (stubSvc) GetByID(ctx context.Context, id int64) (*service.Message, error) {
	return &service.Message{ID: id, Body: "ok"}, nil
}
func (stubSvc) ListByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*service.Message, error) {
	return nil, nil
}
func (stubSvc) Broadcast(ctx context.Context, fromID int64, toIDs []int64, body string) (int, error) {
	return len(toIDs), nil
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

// TestMessages_NoToken_401 验证 /api/v1/messages 无 token → 11001。
func TestMessages_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestMessages_InvalidToken_401 验证非法 token → 11001。
func TestMessages_InvalidToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestSend_WithToken_OK 验证带 token 时 POST /api/v1/messages 走通。
func TestSend_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "patient")
	body := `{"order_id":100,"to_user_id":2,"body":"hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestDetail_WithToken_OK 验证 GET /api/v1/messages/:id。
func TestDetail_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "patient")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/7", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestBroadcast_WithToken_OK 验证 POST /api/v1/messages/broadcast。
func TestBroadcast_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret, nil)
	tok := signToken(t, 1, "super_admin")
	body := `{"to_user_ids":[2,3,4],"body":"系统通知"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/broadcast", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}
