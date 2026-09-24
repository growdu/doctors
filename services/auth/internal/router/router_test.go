package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// stubRepo / stubSMS / stubWX / stubRN 复用 handler_test 中的 fake 不行（不同包）；
// 这里写最小可用 fake。
type stubRepo struct{}

func (r *stubRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	return 1, nil
}
func (r *stubRepo) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	return &service.User{ID: 1, Phone: phone, Role: "patient"}, nil
}
func (r *stubRepo) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	return &service.User{ID: 1, Role: "patient"}, nil
}
func (r *stubRepo) FindByID(ctx context.Context, id int64) (*service.User, error) {
	return &service.User{ID: id, Phone: "13800138000", Role: "patient"}, nil
}
func (r *stubRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return nil
}

type stubSMS struct{}

func (s *stubSMS) Send(ctx context.Context, phone, code string) error { return nil }
func (s *stubSMS) VerifyCode(phone, code string) bool                 { return true }

type stubWX struct{}

func (w *stubWX) Code2Session(ctx context.Context, code string) (string, string, error) {
	return "u-" + code, "o-" + code, nil
}

type stubRN struct{}

func (r *stubRN) Verify(ctx context.Context, name, idCard string) (bool, string, string, error) {
	return true, "h", "1234", nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.New(&stubRepo{}, &stubSMS{}, &stubWX{}, &stubRN{}, "secret", 60_000_000_000)
	return New(handler.New(svc), "secret")
}

// TestHealthz_ReturnsOK /healthz 直返 200 + code=0。
func TestHealthz_ReturnsOK(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var resp httpx.Resp[map[string]string]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Data["status"])
}

// TestRouter_RegistersAllRoutes 验证六个 API 路由都被注册。
func TestRouter_RegistersAllRoutes(t *testing.T) {
	r := newRouter()
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/sms/send"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/users/real-name/auth"},
		{http.MethodGet, "/api/v1/users/me"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"路由 %s %s 应被注册", tc.method, tc.path)
		})
	}
}

// TestRouter_RejectsUnknownPath 验证未注册路径返回 404。
func TestRouter_RejectsUnknownPath(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}