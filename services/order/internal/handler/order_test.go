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
	"github.com/growdu/doctors/services/order/internal/middleware"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/service"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- fake repo ----------

type fakeRepo struct {
	orders map[int64]*repo.Order
	events []*repo.OrderEvent
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{orders: map[int64]*repo.Order{}}
}

func (r *fakeRepo) Create(ctx context.Context, o *repo.Order) error {
	r.nextID++
	o.ID = r.nextID
	r.orders[o.ID] = o
	return nil
}

func (r *fakeRepo) FindByID(ctx context.Context, id int64) (*repo.Order, error) {
	if o, ok := r.orders[id]; ok {
		return o, nil
	}
	return nil, repo.ErrOrderNotFound
}

func (r *fakeRepo) ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.PatientID == patientID {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, to string, v int, escortID *int64) error {
	o, ok := r.orders[id]
	if !ok {
		return errors.New("not found")
	}
	if o.Version != v {
		return repo.ErrVersionConflict
	}
	o.Status = to
	o.Version = v + 1
	if escortID != nil {
		o.EscortID = escortID
	}
	return nil
}

func (r *fakeRepo) InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error {
	r.events = append(r.events, &repo.OrderEvent{
		OrderID: orderID, FromStatus: from, ToStatus: to, ActorID: actorID, Payload: payload,
	})
	return nil
}

func (r *fakeRepo) ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error) {
	return r.events, nil
}

// v1.1：fake 实现选人 + 30s 确认窗口四件套（handler 测试用不到，仅满足接口）。
func (r *fakeRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
	return repo.ErrVersionConflict
}
func (r *fakeRepo) ConfirmByEscort(ctx context.Context, id int64, escortID int64, now time.Time, expectVersion int) error {
	return repo.ErrVersionConflict
}
func (r *fakeRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
	return repo.ErrVersionConflict
}
func (r *fakeRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
	return nil, nil
}

type fakeUserLookup struct {
	users map[int64]*service.UserSnapshot
}

func (u *fakeUserLookup) FindByID(ctx context.Context, id int64) (*service.UserSnapshot, error) {
	if s, ok := u.users[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

// ---------- 装配 ----------

const (
	testSecret = "test-secret"
	testTTL    = 60_000_000_000
)

func newTestServer() (*gin.Engine, *fakeRepo, *service.Service) {
	gin.SetMode(gin.TestMode)
	users := &fakeUserLookup{users: map[int64]*service.UserSnapshot{
		1: {ID: 1, Role: "patient", RealNameVerified: true},
		2: {ID: 2, Role: "escort", RealNameVerified: true},
	}}
	fr := newFakeRepo()
	svc := service.New(fr, users)
	h := New(svc)
	r := gin.New()
	v1 := r.Group("/api/v1", middleware.Auth(testSecret))
	h.RegisterRoutes(v1)
	return r, fr, svc
}

func signTestToken(t *testing.T, uid int64, role string) string {
	t.Helper()
	now := time.Now()
	tok, err := authpkg.Sign(testSecret, authpkg.Claims{
		UserID: uid,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(testTTL)),
		},
	})
	require.NoError(t, err)
	return tok
}

func doRequest(t *testing.T, r *gin.Engine, method, path, token string, body any) *httpx.Resp[map[string]any] {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return &resp
}

// ---------- 测试 ----------

func TestCreate_OK(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders", tok, map[string]any{
		"hospital_id":      100,
		"package_id":       1,
		"service_start_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"amount":           200.0,
	})
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.NotZero(t, resp.Data["id"])
}

func TestCreate_NotRealName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	users := &fakeUserLookup{users: map[int64]*service.UserSnapshot{
		1: {ID: 1, Role: "patient", RealNameVerified: false},
	}}
	fr := newFakeRepo()
	svc := service.New(fr, users)
	h := New(svc)
	r := gin.New()
	v1 := r.Group("/api/v1", middleware.Auth(testSecret))
	h.RegisterRoutes(v1)

	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders", tok, map[string]any{
		"hospital_id":      100,
		"package_id":       1,
		"service_start_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"amount":           200.0,
	})
	assert.NotEqual(t, 0, resp.Code)
}

func TestCreate_BadJSON(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders", tok, map[string]any{})
	assert.NotEqual(t, 0, resp.Code)
}

func TestList_OK(t *testing.T) {
	r, fr, _ := newTestServer()
	// 先建一笔
	fr.Create(context.Background(), &repo.Order{PatientID: 1, Status: "created", OrderNo: "X"})
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders", tok, nil)
	assert.Equal(t, 0, resp.Code)
}

func TestGet_NotFound(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders/9999", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

func TestGet_BadID(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	resp := doRequest(t, r, http.MethodGet, "/api/v1/orders/abc", tok, nil)
	assert.NotEqual(t, 0, resp.Code)
}

func TestAccept_NonEscort(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 1, "patient")
	// v1.1：accept 路由已删除；改测 confirm-accept 路由：patient 不应能 confirm-accept。
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/confirm-accept", tok, nil)
	assert.NotEqual(t, 0, resp.Code, "patient 不应能 confirm-accept")
}

func TestSelectEscort_NonPatient(t *testing.T) {
	r, _, _ := newTestServer()
	tok := signTestToken(t, 2, "escort")
	resp := doRequest(t, r, http.MethodPost, "/api/v1/orders/1/select-escort", tok, map[string]any{"escort_id": 5})
	assert.NotEqual(t, 0, resp.Code, "escort 不应能 select-escort")
}