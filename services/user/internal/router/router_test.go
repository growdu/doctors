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

	"github.com/growdu/doctors/services/user/internal/handler"
	"github.com/growdu/doctors/services/user/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// fakeRepo 满足 service.ProfileRepo。
type fakeRepo struct{}

func (fakeRepo) GetByID(ctx context.Context, id int64) (*service.Profile, error) {
	return &service.Profile{ID: id, Nickname: "alice"}, nil
}
func (fakeRepo) UpdateNickname(ctx context.Context, id int64, n string) error { return nil }
func (fakeRepo) UpdateAvatar(ctx context.Context, id int64, u string) error  { return nil }

func newRouter(r service.ProfileRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.New(r)
	h := handler.New(svc)
	return New(h, "secret")
}

// TestHealthz_ReturnsOK 验证 /healthz 直返 200。
func TestHealthz_ReturnsOK(t *testing.T) {
	r := newRouter(fakeRepo{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]string]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestUserRoutesRequireAuth 验证所有 user 路由必须带 Bearer。
func TestUserRoutesRequireAuth(t *testing.T) {
	r := newRouter(fakeRepo{})
	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/1"},
		{http.MethodPatch, "/api/v1/users/1/nickname"},
		{http.MethodPatch, "/api/v1/users/1/avatar"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			var resp httpx.Resp[map[string]any]
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.NotEqual(t, 0, resp.Code)
		})
	}
}

// TestUserRoutes_404 fallback 中间件返回 404。
func TestUserRoutes_404(t *testing.T) {
	r := newRouter(fakeRepo{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}