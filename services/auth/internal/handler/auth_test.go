package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/services/auth/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- 全套 fake（满足 service 各接口） ----------

type fakeRepo struct{ notFound bool }

func (r *fakeRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	return 1, nil
}
func (r *fakeRepo) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	if r.notFound {
		return nil, errors.New("fake: not found")
	}
	return &service.User{ID: 1, Phone: phone, Role: "patient"}, nil
}
func (r *fakeRepo) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	if r.notFound {
		return nil, errors.New("fake: not found")
	}
	return &service.User{ID: 1, Role: "patient"}, nil
}
func (r *fakeRepo) FindByID(ctx context.Context, id int64) (*service.User, error) {
	if r.notFound {
		return nil, errors.New("fake: not found")
	}
	return &service.User{ID: id, Phone: "13800138000", Role: "patient"}, nil
}
func (r *fakeRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return nil
}

type fakeSMS struct{}

func (s *fakeSMS) Send(ctx context.Context, phone, code string) error { return nil }
func (s *fakeSMS) VerifyCode(phone, code string) bool                 { return code != "" }

type fakeWX struct{}

func (w *fakeWX) Code2Session(ctx context.Context, code string) (string, string, error) {
	return "unionid-" + code, "openid-" + code, nil
}

type fakeRealName struct{}

func (r *fakeRealName) Verify(ctx context.Context, name, idCard string) (bool, string, string, error) {
	return true, "h", "1234", nil
}

const (
	testJWTSecret = "test-secret"
	testJWTTTL    = 60_000_000_000
)

// ---------- 装配 ----------

func newTestServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	svc := service.New(repo, &fakeSMS{}, &fakeWX{}, &fakeRealName{}, testJWTSecret, testJWTTTL)
	h := New(svc)
	r := gin.New()
	h.RegisterRoutes(r, testJWTSecret)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path string, body any) *httpx.Resp[map[string]any] {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

// ---------- 测试 ----------

func TestSendSMS_OK(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/sms/send",
		map[string]string{"phone": "13800138000"})
	assert.Equal(t, 0, resp.Code, resp.Message)
}

func TestSendSMS_InvalidPhone(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/sms/send",
		map[string]string{"phone": "abc"})
	assert.NotEqual(t, 0, resp.Code)
}

func TestLoginSMS_OK(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"type": "sms", "phone": "13800138000", "code": "1234"})
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.NotEmpty(t, resp.Data["token"])
	assert.NotZero(t, resp.Data["user_id"])
}

func TestLoginWX_OK(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"type": "wx", "wx_code": "user-A"})
	assert.Equal(t, 0, resp.Code, resp.Message)
}

func TestLogin_UnknownType(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"type": "unknown"})
	assert.NotEqual(t, 0, resp.Code)
}

func TestRefresh_OK(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"token": "fake-but-handled"})
	// service.Refresh 会因为签名错误返回 401；这里仅检查路由没 404
	assert.NotEqual(t, 0, resp.Code) // fake token 必失败但路由生效
}

func TestRealName_RequiresAuth(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/users/real-name/auth",
		map[string]string{"name": "张三", "id_card": "110101199001011234"})
	assert.NotEqual(t, 0, resp.Code, "no token 必须被拒")
}

func TestMe_RequiresAuth(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodGet, "/api/v1/users/me", nil)
	assert.NotEqual(t, 0, resp.Code)
}

func TestMe_OK_WithToken(t *testing.T) {
	// 用合法 jwt（来自共享包）
	tok, err := signTestToken(t, 99, "patient")
	require.NoError(t, err)
	r := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.EqualValues(t, 99, resp.Data["id"])
}

// signTestToken 直接用 shared/auth 签发 JWT，避免重复实现。
func signTestToken(t *testing.T, uid int64, role string) (string, error) {
	t.Helper()
	now := time.Now()
	return authpkg.Sign(testJWTSecret, authpkg.Claims{
		UserID: uid,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(testJWTTTL)),
		},
	})
}

// signTestTokenMulti 签多角色 token（v2）。
func signTestTokenMulti(t *testing.T, uid int64, roles []string, active string) (string, error) {
	t.Helper()
	now := time.Now()
	return authpkg.Sign(testJWTSecret, authpkg.Claims{
		UserID:  uid,
		Roles:   roles,
		Active:  active,
		Role:    active, // v1 兼容
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(testJWTTTL)),
		},
	})
}

// TestSwitchRole_OK 验证多角色 token 切换 active 成功。
func TestSwitchRole_OK(t *testing.T) {
	tok, err := signTestTokenMulti(t, 100, []string{"patient", "escort"}, "patient")
	require.NoError(t, err)
	r := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-role",
		bytes.NewBufferString(`{"active":"escort"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.NotEmpty(t, resp.Data["token"], "应返回新 token")
	assert.Equal(t, "escort", resp.Data["active"])
	assert.EqualValues(t, 100, resp.Data["user_id"])
	roles, _ := resp.Data["roles"].([]any)
	assert.Len(t, roles, 2)
}

// TestSwitchRole_RequiresAuth 验证无 token 必须 401。
func TestSwitchRole_RequiresAuth(t *testing.T) {
	r := newTestServer()
	resp := doRequest(t, r, http.MethodPost, "/api/v1/auth/switch-role",
		map[string]string{"active": "escort"})
	assert.NotEqual(t, 0, resp.Code, "无 token 业务码应 != 0")
}

// TestSwitchRole_ActiveNotInRoles 验证 active 不在 roles 中返 400。
func TestSwitchRole_ActiveNotInRoles(t *testing.T) {
	tok, err := signTestTokenMulti(t, 100, []string{"patient"}, "patient")
	require.NoError(t, err)
	r := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-role",
		bytes.NewBufferString(`{"active":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code, "active 不在 roles 应失败")
	assert.Contains(t, resp.Message, "active")
}

// TestSwitchRole_MissingActive 验证缺 active 字段返 400。
func TestSwitchRole_MissingActive(t *testing.T) {
	tok, err := signTestTokenMulti(t, 100, []string{"patient", "escort"}, "patient")
	require.NoError(t, err)
	r := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-role",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}