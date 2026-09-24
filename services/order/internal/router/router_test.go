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

	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// stubOrderRepo 满足 service.OrderRepo；handler 测试不触达业务逻辑。
type stubOrderRepo struct{}

func (stubOrderRepo) Create(ctx context.Context, o *repo.Order) error               { return nil }
func (stubOrderRepo) FindByID(ctx context.Context, id int64) (*repo.Order, error)  { return nil, repo.ErrOrderNotFound }
func (stubOrderRepo) ListByPatient(ctx context.Context, id int64, l, o int) ([]*repo.Order, error) {
	return nil, nil
}
func (stubOrderRepo) UpdateStatus(ctx context.Context, id int64, to string, v int, e *int64) error {
	return nil
}
func (stubOrderRepo) InsertEvent(ctx context.Context, id int64, from *string, to string, a *int64, p []byte) error {
	return nil
}
func (stubOrderRepo) ListEvents(ctx context.Context, id int64) ([]*repo.OrderEvent, error) {
	return nil, nil
}

type stubUserLookup struct{}

func (stubUserLookup) FindByID(ctx context.Context, id int64) (*service.UserSnapshot, error) {
	return &service.UserSnapshot{ID: id, Role: "patient", RealNameVerified: true}, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.New(stubOrderRepo{}, stubUserLookup{})
	return New(handler.New(svc), "secret")
}

// TestHealthz_ReturnsOK /healthz 直返 200。
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

// TestRouter_RegistersOrderRoutes 验证六个 API 路由都被注册。
func TestRouter_RegistersOrderRoutes(t *testing.T) {
	r := newRouter()
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/orders"},
		{http.MethodGet, "/api/v1/orders"},
		{http.MethodGet, "/api/v1/orders/1"},
		{http.MethodPost, "/api/v1/orders/1/accept"},
		{http.MethodPost, "/api/v1/orders/1/cancel"},
		{http.MethodPost, "/api/v1/orders/1/finish"},
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

// TestRouter_OrdersRequireAuth 验证 /api/v1/orders/* 必须带 Bearer。
func TestRouter_OrdersRequireAuth(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code, "无 token 应被 401 拦截")
}