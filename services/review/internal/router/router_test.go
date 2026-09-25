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

	"github.com/growdu/doctors/services/review/internal/handler"
	"github.com/growdu/doctors/services/review/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "review-router-secret"

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct{}

func (stubSvc) CreateReview(ctx context.Context, reviewerID, orderID, escortID int64, rating int, comment string) (*service.Review, error) {
	return &service.Review{ID: 1, OrderID: orderID, EscortID: escortID, Rating: rating}, nil
}
func (stubSvc) GetByID(ctx context.Context, id int64) (*service.Review, error) {
	return &service.Review{ID: id, Rating: 5}, nil
}
func (stubSvc) List(ctx context.Context, f service.ListFilter) ([]*service.Review, error) {
	return nil, nil
}
func (stubSvc) Reply(ctx context.Context, id, adminID int64, body string) (*service.Review, error) {
	return &service.Review{ID: id, Reply: body, RepliedBy: adminID}, nil
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
	r := New(handler.New(stubSvc{}), testSecret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, 200, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestReviews_NoToken_401 验证 /api/v1/reviews 无 token → 11001。
func TestReviews_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestReviews_InvalidToken_401 验证非法 token → 11001。
func TestReviews_InvalidToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 11001, resp.Code)
}

// TestCreate_WithToken_OK 验证 POST /api/v1/reviews 带 token 走通。
func TestCreate_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret)
	tok := signToken(t, 1, "patient")
	body := `{"order_id":100,"escort_id":7,"rating":5,"comment":"好"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestDetail_WithToken_OK 验证 GET /api/v1/reviews/:id。
func TestDetail_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret)
	tok := signToken(t, 1, "patient")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/7", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestReply_WithToken_OK 验证 POST /api/v1/reviews/:id/reply。
func TestReply_WithToken_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(handler.New(stubSvc{}), testSecret)
	tok := signToken(t, 1, "super_admin")
	body := `{"body":"感谢反馈"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/7/reply", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}
