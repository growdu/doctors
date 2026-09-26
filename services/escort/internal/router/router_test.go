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

	"github.com/growdu/doctors/services/escort/internal/availability"
	"github.com/growdu/doctors/services/escort/internal/handler"
	"github.com/growdu/doctors/services/escort/internal/service"
	authpkg "github.com/growdu/doctors/shared/auth"
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
func (r *fakeRepo) UpdateCity(ctx context.Context, id int64, city string) error {
	if e, ok := r.byID[id]; ok {
		e.City = city
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

// fakeAvailRepo 满足 availability.AvailabilityRepo。
type fakeAvailRepo struct{}

func (fakeAvailRepo) Create(ctx context.Context, escortID int64, startAt, endAt time.Time) (*availability.Availability, error) {
	return &availability.Availability{ID: 1, EscortID: escortID, StartAt: startAt, EndAt: endAt, Status: availability.StatusAvailable}, nil
}
func (fakeAvailRepo) FindByID(ctx context.Context, id int64) (*availability.Availability, error) {
	return nil, availability.ErrAvailabilityNotFound
}
func (fakeAvailRepo) Delete(ctx context.Context, id, escortID int64) error { return nil }
func (fakeAvailRepo) ListByEscort(ctx context.Context, escortID int64) ([]*availability.Availability, error) {
	return nil, nil
}
func (fakeAvailRepo) ListAvailableByTime(ctx context.Context, startAt, endAt time.Time, limit int) ([]*availability.Availability, error) {
	return nil, nil
}
func (fakeAvailRepo) BookByOrder(ctx context.Context, id, orderID int64) error   { return nil }
func (fakeAvailRepo) ReleaseByOrder(ctx context.Context, orderID int64) error    { return nil }

const testSecret = "escort-router-secret"

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{byID: map[int64]*service.Escort{}, byUserID: map[int64]int64{}}
	svc := service.New(repo, fakePub{})
	availSvc := availability.NewService(fakeAvailRepo{})
	availH := availability.NewHandler(availSvc)
	return NewWithPublic(handler.New(svc), availH, testSecret, nil)
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
		{http.MethodPatch, "/api/v1/escorts/me/location"},
		{http.MethodPatch, "/api/v1/escorts/me/city"},
		{http.MethodPost, "/api/v1/escorts/me/qualifications"},
		{http.MethodGet, "/api/v1/escorts/me/qualifications"},
		{http.MethodPost, "/api/v1/escorts/me/trainings"},
		{http.MethodGet, "/api/v1/escorts/me/trainings"},
		// availability subpkg (authed)
		{http.MethodPut, "/api/v1/escorts/me/availability"},
		{http.MethodGet, "/api/v1/escorts/me/availability"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(""))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			var resp httpx.Resp[map[string]any]
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.NotEqual(t, 0, resp.Code)
		})
	}
}

// TestPublicAvailabilityList_NoAuth 验证公开 availability 端点不挂 auth。
func TestPublicAvailabilityList_NoAuth(t *testing.T) {
	r := newRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		"/api/v1/escorts/1/availabilities?start_at=2026-09-24T00:00:00Z&end_at=2026-09-25T00:00:00Z", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
}

// TestMeRoutesWithToken 验证带 token 时部分 /me 路由不报 401（service 内部会报其它业务码）。
func TestMeRoutesWithToken(t *testing.T) {
	r := newRouter()
	tok := signToken(t, 1, "escort")
	cases := []struct{ method, path string }{
		{http.MethodPatch, "/api/v1/escorts/me/location"},
		{http.MethodPatch, "/api/v1/escorts/me/city"},
		{http.MethodGet, "/api/v1/escorts/me/qualifications"},
		{http.MethodGet, "/api/v1/escorts/me/trainings"},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			var body *strings.Reader
			if tc.method == http.MethodPatch {
				body = strings.NewReader(`{}`)
			}
			var req *http.Request
			if body != nil {
				req = httptest.NewRequest(tc.method, tc.path, body)
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			req.Header.Set("Authorization", "Bearer "+tok)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			var resp httpx.Resp[map[string]any]
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			// 注：service 内业务校验会返错；但 jwt 已通过，code != 11001
			assert.NotEqual(t, 11001, resp.Code)
		})
	}
}
