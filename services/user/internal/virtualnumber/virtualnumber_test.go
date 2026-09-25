package virtualnumber

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- fake repo ----------

type fakeRepo struct {
	mu     sync.Mutex
	byID   map[int64]*Record
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*Record{}, nextID: 1}
}

func (r *fakeRepo) Allocate(ctx context.Context, in AllocateInput) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range r.byID {
		if v.OrderID == in.OrderID && v.Status == "active" {
			return nil, ErrDuplicateActive
		}
	}
	v := &Record{
		ID:        r.nextID,
		OrderID:   in.OrderID,
		PatientID: in.PatientID,
		EscortID:  in.EscortID,
		Phone:     in.Phone,
		Status:    "active",
		ExpireAt:  in.ExpireAt,
		CreatedAt: time.Now(),
	}
	if v.Phone == "" {
		v.Phone = "17000000001"
	}
	r.nextID++
	r.byID[v.ID] = v
	return v, nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	v.Status = deriveStatus(v.Status, v.ReleasedAt, v.ExpireAt, time.Now())
	return v, nil
}

func (r *fakeRepo) Release(ctx context.Context, id int64, reason string) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byID[id]
	if !ok || v.Status != "active" {
		return nil, ErrNotFound
	}
	v.Status = "released"
	now := time.Now()
	v.ReleasedAt = &now
	return v, nil
}

// ---------- service tests ----------

func TestAllocate_OK(t *testing.T) {
	s, _ := newSvc()
	v, err := s.Allocate(context.Background(), AllocateRequest{
		OrderID:   100,
		PatientID: 1,
		EscortID:  2,
		ExpireAt:  time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	assert.NotZero(t, v.ID)
	assert.NotEmpty(t, v.Phone)
	assert.Equal(t, "active", v.Status)
}

func TestAllocate_Invalid(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Allocate(context.Background(), AllocateRequest{
		OrderID:   0,
		PatientID: 1,
		EscortID:  2,
		ExpireAt:  time.Now().Add(time.Hour),
	})
	var e *errs.Error
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeParamInvalid, e.Code)

	_, err = s.Allocate(context.Background(), AllocateRequest{
		OrderID:   1,
		PatientID: 1,
		EscortID:  2,
		ExpireAt:  time.Now().Add(-time.Hour),
	})
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeParamInvalid, e.Code)
}

func TestAllocate_Duplicate(t *testing.T) {
	s, _ := newSvc()
	in := AllocateRequest{
		OrderID:   100,
		PatientID: 1,
		EscortID:  2,
		ExpireAt:  time.Now().Add(time.Hour),
	}
	_, err := s.Allocate(context.Background(), in)
	require.NoError(t, err)
	_, err = s.Allocate(context.Background(), in)
	var e *errs.Error
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeConflict, e.Code)
}

func TestGet_OK(t *testing.T) {
	s, _ := newSvc()
	v, _ := s.Allocate(context.Background(), AllocateRequest{
		OrderID: 1, PatientID: 1, EscortID: 2,
		ExpireAt: time.Now().Add(time.Hour),
	})
	got, err := s.Get(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, v.ID, got.ID)
}

func TestGet_NotFound(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Get(context.Background(), 9999)
	var e *errs.Error
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

func TestGet_RederiveExpired(t *testing.T) {
	s, repo := newSvc()
	v, _ := s.Allocate(context.Background(), AllocateRequest{
		OrderID: 1, PatientID: 1, EscortID: 2,
		ExpireAt: time.Now().Add(time.Hour),
	})
	repo.mu.Lock()
	repo.byID[v.ID].ExpireAt = time.Now().Add(-time.Hour)
	repo.mu.Unlock()

	got, err := s.Get(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, "expired", got.Status)
}

// ---------- handler tests ----------

func buildRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	h := NewHandler(svc)
	h.RegisterRoutes(g)
	return r
}

func doReq(t *testing.T, r *gin.Engine, method, path, body string) (int, httpx.Resp[map[string]any]) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func TestHandler_Allocate_OK(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	body := `{"order_id":100,"patient_id":1,"escort_id":2,"expire_at":"` +
		time.Now().Add(time.Hour).Format(time.RFC3339) + `"}`
	code, resp := doReq(t, r, http.MethodPost, "/api/v1/virtual-numbers/allocate", body)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Allocate_BadExpire(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	body := `{"order_id":100,"patient_id":1,"escort_id":2,"expire_at":"` +
		time.Now().Add(-time.Hour).Format(time.RFC3339) + `"}`
	code, resp := doReq(t, r, http.MethodPost, "/api/v1/virtual-numbers/allocate", body)
	assert.Equal(t, http.StatusOK, code)
	assert.NotEqual(t, 0, resp.Code)
}

func TestHandler_Get_OK(t *testing.T) {
	s, _ := newSvc()
	v, _ := s.Allocate(context.Background(), AllocateRequest{
		OrderID: 1, PatientID: 1, EscortID: 2,
		ExpireAt: time.Now().Add(time.Hour),
	})
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/virtual-numbers/"+itoa(v.ID), "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

// ---------- helpers ----------

func newSvc() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo), repo
}

func errorsAs(err error, e **errs.Error) bool {
	if err == nil {
		return false
	}
	if v, ok := errs.As(err); ok {
		*e = v
		return true
	}
	return false
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}