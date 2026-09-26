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

	"github.com/growdu/doctors/services/user/internal/address"
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

// fakeAddrRepo 给 address.Service 用。
type fakeAddrRepo struct{}

func (fakeAddrRepo) Create(ctx context.Context, a *address.Record) error    { return nil }
func (fakeAddrRepo) CreateDefault(ctx context.Context, a *address.Record) error { return nil }
func (fakeAddrRepo) ListByUser(ctx context.Context, userID int64) ([]*address.Record, error) {
	return nil, nil
}
func (fakeAddrRepo) CountByUser(ctx context.Context, userID int64) (int, error) { return 0, nil }
func (fakeAddrRepo) GetByID(ctx context.Context, id, userID int64) (*address.Record, error) {
	return nil, address.ErrNotFound
}
func (fakeAddrRepo) Update(ctx context.Context, a *address.Record) error       { return nil }
func (fakeAddrRepo) SetDefault(ctx context.Context, id, userID int64) error   { return nil }
func (fakeAddrRepo) Delete(ctx context.Context, id, userID int64) error       { return nil }

func newRouter(r service.ProfileRepo, asvc *address.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.New(r)
	h := handler.New(svc)
	return New(Deps{ProfileSvc: svc, AddressSvc: asvc}, h, "secret", nil)
}

// TestHealthz_ReturnsOK 验证 /healthz 直返 200。
func TestHealthz_ReturnsOK(t *testing.T) {
	r := newRouter(fakeRepo{}, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]string]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestUserRoutesRequireAuth 验证所有 user 路由必须带 Bearer。
func TestUserRoutesRequireAuth(t *testing.T) {
	r := newRouter(fakeRepo{}, nil)
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
	r := newRouter(fakeRepo{}, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAddressRoutes_RequireAuth 验证地址路由也要 Bearer。
func TestAddressRoutes_RequireAuth(t *testing.T) {
	asvc := address.NewService(fakeAddrRepo{})
	r := newRouter(fakeRepo{}, asvc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/addresses", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

// TestAddressRoutes_NilSvc_NoRoutes 验证 AddressSvc=nil 时不挂地址路由。
func TestAddressRoutes_NilSvc_NoRoutes(t *testing.T) {
	r := newRouter(fakeRepo{}, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/addresses", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}