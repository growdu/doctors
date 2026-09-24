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

	"github.com/growdu/doctors/services/match/internal/handler"
	"github.com/growdu/doctors/services/match/internal/pool"
	"github.com/growdu/doctors/services/match/internal/scorer"
	"github.com/growdu/doctors/services/match/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// stubLoader 返回固定 escort list。
type stubLoader struct{}

func (stubLoader) ListAvailable(ctx context.Context, city string) ([]scorer.Escort, error) {
	return nil, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.New(pool.NewNopPool(), stubLoader{}, 0)
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
}

// TestRouter_RegistersMatchRoutes 验证 3 个 API 路由都注册。
func TestRouter_RegistersMatchRoutes(t *testing.T) {
	r := newRouter()
	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/match/feed"},
		{http.MethodPost, "/api/v1/match/candidates"},
		{http.MethodPost, "/internal/match/dispatch"},
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

// TestRouter_MatchRequireAuth 验证 /api/v1/match/* 必须带 Bearer。
func TestRouter_MatchRequireAuth(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/match/feed", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code, "无 token 应被 401 拦截")
}