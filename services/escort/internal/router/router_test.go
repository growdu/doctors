package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// fakeRepo 满足 service.EscortRepo。
type fakeRepo struct {
	byID     map[int64]*service.Escort
	byUserID map[int64]int64
}

func (r *fakeRepo) Create(ctx context.Context, e *service.Escort) error {
	r.byID[e.ID] = e
	r.byUserID[e.UserID] = e.ID
	return nil
}
func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*service.Escort, error) {
	if e, ok := r.byID[id]; ok {
		return e, nil
	}
	return nil, service.ErrEscortNotFound
}
func (r *fakeRepo) GetByUserID(ctx context.Context, uid int64) (*service.Escort, error) {
	id, ok := r.byUserID[uid]
	if !ok {
		return nil, service.ErrEscortNotFound
	}
	return r.byID[id], nil
}
func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, s string) error {
	if e, ok := r.byID[id]; ok {
		e.Status = s
	}
	return nil
}
func (r *fakeRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	if e, ok := r.byID[id]; ok {
		e.Lat, e.Lng = lat, lng
	}
	return nil
}
func (r *fakeRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	if e, ok := r.byID[id]; ok {
		e.AvailableFrom, e.AvailableUntil = from, until
	}
	return nil
}

type fakePub struct{}

func (fakePub) PublishAvailabilityChanged(ctx context.Context, ev service.AvailabilityEvent) error {
	return nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{byID: map[int64]*service.Escort{}, byUserID: map[int64]int64{}}
	svc := service.New(repo, fakePub{})
	return New(handler.New(svc), "secret")
}

// TestHealthz_ReturnsOK 验证 /healthz。
func TestHealthz_ReturnsOK(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]string]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestEscortRoutesRequireAuth 验证 escort 路由都要求 Bearer。
func TestEscortRoutesRequireAuth(t *testing.T) {
	r := newRouter()
	cases := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/escorts"},
		{http.MethodGet, "/api/v1/escorts/1"},
		{http.MethodPost, "/api/v1/escorts/1/availability"},
		{http.MethodPatch, "/api/v1/escorts/1/location"},
		{http.MethodPatch, "/api/v1/escorts/1/city"},
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