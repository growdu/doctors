package pkg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// ---------- fake repo ----------

type fakeRepo struct {
	mu       sync.Mutex
	packages map[int64]*Record
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{packages: map[int64]*Record{}, nextID: 1}
}

func (r *fakeRepo) ListByHospital(ctx context.Context, hospitalID int64) ([]*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*Record{}
	for _, p := range r.packages {
		if p.HospitalID == hospitalID && p.Status == "active" {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.packages[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

// ---------- service tests ----------

func seedPackage(s *Service, repo *fakeRepo, hospitalID int64, name, typ string, duration int, price float64) *Record {
	p := &Record{
		ID:          repo.nextID,
		HospitalID:  hospitalID,
		Name:        name,
		Type:        typ,
		DurationMin: duration,
		Price:       price,
		Status:      "active",
	}
	repo.nextID++
	repo.packages[p.ID] = p
	return p
}

func TestListByHospital_OK(t *testing.T) {
	s, repo := newSvc()
	seedPackage(s, repo, 1, "半日陪诊", "half_day", 240, 200)
	seedPackage(s, repo, 1, "全日陪诊", "full_day", 480, 380)
	seedPackage(s, repo, 2, "单项陪诊", "single_item", 60, 80)
	list, err := s.ListByHospital(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListByHospital_Empty(t *testing.T) {
	s, _ := newSvc()
	list, err := s.ListByHospital(context.Background(), 999)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestListByHospital_BadHospitalID(t *testing.T) {
	s, _ := newSvc()
	_, err := s.ListByHospital(context.Background(), 0)
	var e *errs.Error
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeParamInvalid, e.Code)
}

func TestGet_OK(t *testing.T) {
	s, repo := newSvc()
	p := seedPackage(s, repo, 1, "x", "half_day", 240, 200)
	got, err := s.Get(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, "x", got.Name)
}

func TestGet_NotFound(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Get(context.Background(), 9999)
	var e *errs.Error
	require.True(t, errorsAs(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
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

func TestHandler_ListByHospital_OK(t *testing.T) {
	s, repo := newSvc()
	seedPackage(s, repo, 1, "x", "half_day", 240, 200)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/hospitals/1/packages", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_ListByHospital_BadID(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/hospitals/abc/packages", "")
	assert.Equal(t, http.StatusOK, code)
	assert.NotEqual(t, 0, resp.Code)
}

func TestHandler_Get_OK(t *testing.T) {
	s, repo := newSvc()
	p := seedPackage(s, repo, 1, "x", "half_day", 240, 200)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/packages/"+itoa(p.ID), "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/packages/9999", "")
	assert.Equal(t, http.StatusOK, code)
	assert.NotEqual(t, 0, resp.Code)
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