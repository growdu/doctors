package availability

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

	mw "github.com/growdu/doctors/services/escort/internal/middleware"
)

const (
	testSecret = "test-secret"
	testTTL    = 60_000_000_000
)

// signTestToken 用 testSecret 签发 JWT；role 注入 token claims。
// 与 auth.Sign 对齐：Claims.UserID 字段对应 JSON key "uid"。
func signTestToken(t *testing.T, uid int64, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"uid":  uid,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return s
}

// newTestServer 装配 gin.Engine + handler，附带 escort.Auth 中间件。
func newTestServer(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newServiceWithClock(newFakeRepo(), time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC))
	h := NewHandler(svc)
	r := gin.New()
	api := r.Group("/api/v1")
	authed := api.Group("/", mw.Auth(testSecret))
	h.RegisterRoutes(authed)
	h.RegisterPublicRoutes(api) // 公开路由（无需 token）
	return r, svc
}

// doRequest 通用请求执行器。
func doRequest(t *testing.T, r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestHandler_AddAvailability_OK 验证 escort 加时段。
func TestHandler_AddAvailability_OK(t *testing.T) {
	r, _ := newTestServer(t)
	tok := signTestToken(t, 7, "escort")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	w := doRequest(t, r, http.MethodPut, "/api/v1/escorts/me/availability", tok, gin.H{
		"start_at": now.Add(2 * time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(5 * time.Hour).Format(time.RFC3339),
	})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, _ := resp["data"].(map[string]any)
	assert.Equal(t, "available", data["status"])
	assert.EqualValues(t, 7, data["escort_id"])
}

// TestAddAvailability_NonEscort 验证非 escort role 被拒。
func TestHandler_AddAvailability_NonEscort(t *testing.T) {
	r, _ := newTestServer(t)
	tok := signTestToken(t, 7, "patient")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	w := doRequest(t, r, http.MethodPut, "/api/v1/escorts/me/availability", tok, gin.H{
		"start_at": now.Add(2 * time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(5 * time.Hour).Format(time.RFC3339),
	})
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp["code"], "non-escort role 应被拒")
}

// TestAddAvailability_TimeInvalid 验证 end <= start 被拒。
func TestHandler_AddAvailability_TimeInvalid(t *testing.T) {
	r, _ := newTestServer(t)
	tok := signTestToken(t, 7, "escort")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	w := doRequest(t, r, http.MethodPut, "/api/v1/escorts/me/availability", tok, gin.H{
		"start_at": now.Add(2 * time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(2 * time.Hour).Format(time.RFC3339),
	})
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp["code"], "end<=start 应被拒")
}

// TestAddAvailability_Conflict 验证时段冲突被拒（422）。
func TestHandler_AddAvailability_Conflict(t *testing.T) {
	r, svc := newTestServer(t)
	tok := signTestToken(t, 7, "escort")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	body := gin.H{
		"start_at": now.Add(2 * time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(5 * time.Hour).Format(time.RFC3339),
	}
	// 第一次成功
	w1 := doRequest(t, r, http.MethodPut, "/api/v1/escorts/me/availability", tok, body)
	assert.Equal(t, http.StatusOK, w1.Code)
	// 第二次冲突
	body2 := gin.H{
		"start_at": now.Add(3 * time.Hour).Format(time.RFC3339),
		"end_at":   now.Add(6 * time.Hour).Format(time.RFC3339),
	}
	w2 := doRequest(t, r, http.MethodPut, "/api/v1/escorts/me/availability", tok, body2)
	var resp2 map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	assert.NotEqual(t, 0, resp2["code"], "时段冲突应被拒（业务码非 0）")

	_ = svc // 保留 svc 引用以防 future 用
}

// TestListMine_OK 验证查自己的时段。
func TestHandler_ListMine_OK(t *testing.T) {
	r, svc := newTestServer(t)
	tok := signTestToken(t, 7, "escort")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	_, _ = svc.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(2*time.Hour))
	w := doRequest(t, r, http.MethodGet, "/api/v1/escorts/me/availability", tok, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, _ := resp["data"].(map[string]any)
	items, _ := data["items"].([]any)
	assert.Len(t, items, 1)
}

// TestListByEscort_Public 验证公开查询（无 token）。
// 注：URL 中 + 是 query 值的合法字符，但 RFC3339 时区 +08:00 含 + 时 URL encode 为 %2B 以防被当空格。
func TestHandler_ListByEscort_Public(t *testing.T) {
	r, svc := newTestServer(t)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	_, _ = svc.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(3*time.Hour))
	startURL := now.Add(time.Hour).Format("2006-01-02T15:04:05Z07:00")
	endURL := now.Add(2 * time.Hour).Format("2006-01-02T15:04:05Z07:00")
	w := doRequest(t, r, http.MethodGet, "/api/v1/escorts/7/availabilities?start_at="+startURL+"&end_at="+endURL, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, _ := resp["data"].(map[string]any)
	items, _ := data["items"].([]any)
	assert.GreaterOrEqual(t, len(items), 1)
}

// TestRemoveMine_OK 验证 escort 删自己的时段。
func TestHandler_RemoveMine_OK(t *testing.T) {
	r, svc := newTestServer(t)
	tok := signTestToken(t, 7, "escort")
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	a, err := svc.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(2*time.Hour))
	require.NoError(t, err)
	w := doRequest(t, r, http.MethodDelete, "/api/v1/escorts/me/availability/"+itoa(a.ID), tok, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestRemoveMine_NotOwner 验证 escort A 删 escort B 的时段被拒。
// 注：httpx.Fail 固定 HTTP 200，按 body.code 判定业务码（11002 = Forbidden）。
func TestHandler_RemoveMine_NotOwner(t *testing.T) {
	r, svc := newTestServer(t)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	a, _ := svc.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(2*time.Hour))
	tok := signTestToken(t, 99, "escort")
	w := doRequest(t, r, http.MethodDelete, "/api/v1/escorts/me/availability/"+itoa(a.ID), tok, nil)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.EqualValues(t, 11002, resp["code"], "expected CodeForbidden for not owner")
}

// 简单 itoa 工具。
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// 确保引用了 middleware（避免 unused import 错误）。
var _ = mw.Role