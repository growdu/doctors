package hospital

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
	hospitals map[int64]*Record
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{hospitals: map[int64]*Record{}, nextID: 1}
}

func (r *fakeRepo) List(ctx context.Context, f ListFilter) ([]*Record, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*Record{}
	for _, h := range r.hospitals {
		if f.Status != "" && h.Status != f.Status {
			continue
		}
		if f.CityID > 0 && h.CityID != f.CityID {
			continue
		}
		if f.Level != "" && h.Level != f.Level {
			continue
		}
		if k := f.Keyword; k != "" && !strings.Contains(h.Name, k) {
			continue
		}
		out = append(out, h)
	}
	return out, len(out), nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hospitals[id]
	if !ok {
		return nil, ErrNotFound
	}
	return h, nil
}

// ---------- service tests ----------

func seedHospital(s *Service, repo *fakeRepo, name string, cityID int64, level string) *Record {
	h := &Record{
		ID:      repo.nextID,
		Name:    name,
		CityID:  cityID,
		Level:   level,
		Status:  "active",
		Address: "addr",
	}
	repo.nextID++
	repo.hospitals[h.ID] = h
	return h
}

func TestList_Empty(t *testing.T) {
	s, _ := newSvc()
	out, err := s.List(context.Background(), 1, 20, 0, "", "")
	require.NoError(t, err)
	assert.Equal(t, 0, out.Total)
	assert.Empty(t, out.Items)
}

func TestList_FilterByCity(t *testing.T) {
	s, repo := newSvc()
	seedHospital(s, repo, "协和", 1, "3a")
	seedHospital(s, repo, "301", 1, "3a")
	seedHospital(s, repo, "北医三院", 1, "3a")
	out, err := s.List(context.Background(), 1, 20, 1, "", "")
	require.NoError(t, err)
	assert.Equal(t, 3, out.Total)
}

func TestList_FilterByLevel(t *testing.T) {
	s, repo := newSvc()
	seedHospital(s, repo, "协和", 1, "3a")
	seedHospital(s, repo, "社区医院", 1, "1")
	out, err := s.List(context.Background(), 1, 20, 0, "3a", "")
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
}

func TestList_Keyword(t *testing.T) {
	s, repo := newSvc()
	seedHospital(s, repo, "协和医院", 1, "3a")
	seedHospital(s, repo, "北医三院", 1, "3a")
	out, err := s.List(context.Background(), 1, 20, 0, "", "协和")
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, "协和医院", out.Items[0].Name)
}

func TestList_PaginationDefaults(t *testing.T) {
	s, _ := newSvc()
	out, err := s.List(context.Background(), 0, 0, 0, "", "")
	require.NoError(t, err)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 20, out.Limit)
}

func TestGet_OK(t *testing.T) {
	s, repo := newSvc()
	h := seedHospital(s, repo, "x", 1, "3a")
	got, err := s.Get(context.Background(), h.ID)
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

func TestHandler_List_OK(t *testing.T) {
	s, repo := newSvc()
	seedHospital(s, repo, "x", 1, "3a")
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/hospitals?city_id=1", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Get_OK(t *testing.T) {
	s, repo := newSvc()
	h := seedHospital(s, repo, "x", 1, "3a")
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/hospitals/"+itoa(h.ID), "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/hospitals/9999", "")
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