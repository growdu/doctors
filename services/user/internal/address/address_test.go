package address

import (
	"context"
	"encoding/json"
	"errors"
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
	mu       sync.Mutex
	byID     map[int64]*Record
	nextID   int64
	byUser   map[int64][]int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:   map[int64]*Record{},
		byUser: map[int64][]int64{},
		nextID: 1,
	}
}

func (r *fakeRepo) Create(ctx context.Context, a *Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a.ID = r.nextID
	r.nextID++
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	a.IsDefault = false
	r.byID[a.ID] = a
	r.byUser[a.UserID] = append(r.byUser[a.UserID], a.ID)
	return nil
}

func (r *fakeRepo) CreateDefault(ctx context.Context, a *Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 清掉同用户其他默认
	for _, id := range r.byUser[a.UserID] {
		if r.byID[id].IsDefault {
			r.byID[id].IsDefault = false
		}
	}
	a.ID = r.nextID
	r.nextID++
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	a.IsDefault = true
	r.byID[a.ID] = a
	r.byUser[a.UserID] = append(r.byUser[a.UserID], a.ID)
	return nil
}

func (r *fakeRepo) ListByUser(ctx context.Context, userID int64) ([]*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*Record{}
	for _, id := range r.byUser[userID] {
		out = append(out, r.byID[id])
	}
	return out, nil
}

func (r *fakeRepo) CountByUser(ctx context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byUser[userID]), nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id, userID int64) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok || a.UserID != userID {
		return nil, ErrNotFound
	}
	return a, nil
}

func (r *fakeRepo) Update(ctx context.Context, a *Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old, ok := r.byID[a.ID]
	if !ok || old.UserID != a.UserID {
		return ErrNotFound
	}
	old.Recipient = a.Recipient
	old.Phone = a.Phone
	old.Detail = a.Detail
	old.Lat = a.Lat
	old.Lng = a.Lng
	old.UpdatedAt = time.Now()
	return nil
}

func (r *fakeRepo) SetDefault(ctx context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok || a.UserID != userID {
		return ErrNotFound
	}
	for _, sid := range r.byUser[userID] {
		r.byID[sid].IsDefault = false
	}
	a.IsDefault = true
	return nil
}

func (r *fakeRepo) Delete(ctx context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.byID[id]
	if !ok || a.UserID != userID {
		return ErrNotFound
	}
	delete(r.byID, id)
	list := r.byUser[userID]
	for i, x := range list {
		if x == id {
			r.byUser[userID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	return nil
}

// ---------- service tests ----------

func newSvc() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo), repo
}

func TestService_List_Empty(t *testing.T) {
	s, _ := newSvc()
	list, err := s.List(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestService_Create_OK(t *testing.T) {
	s, repo := newSvc()
	a, err := s.Create(context.Background(), 1, CreateInput{
		Recipient: "张三", Phone: "13800138000", Detail: "中关村大街1号",
	})
	require.NoError(t, err)
	assert.NotZero(t, a.ID)
	assert.False(t, a.IsDefault)

	// repo 计数为 1
	n, _ := repo.CountByUser(context.Background(), 1)
	assert.Equal(t, 1, n)
}

func TestService_Create_Default(t *testing.T) {
	s, repo := newSvc()
	a1, _ := s.Create(context.Background(), 1, CreateInput{Recipient: "甲", Phone: "13800000001", Detail: "a"})
	a2, err := s.Create(context.Background(), 1, CreateInput{Recipient: "乙", Phone: "13800000002", Detail: "b", IsDefault: true})
	require.NoError(t, err)
	assert.True(t, a2.IsDefault)
	assert.False(t, a1.IsDefault)

	list, _ := repo.ListByUser(context.Background(), 1)
	assert.Len(t, list, 2)
}

func TestService_Create_Limit(t *testing.T) {
	s, _ := newSvc()
	for i := 0; i < 5; i++ {
		_, err := s.Create(context.Background(), 1, CreateInput{
			Recipient: "r", Phone: "13800000000", Detail: "d",
		})
		require.NoError(t, err)
	}
	_, err := s.Create(context.Background(), 1, CreateInput{
		Recipient: "r", Phone: "13800000000", Detail: "d",
	})
	require.Error(t, err)
	var e *errs.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errs.CodeConflict, e.Code)
}

func TestService_Create_BadInput(t *testing.T) {
	s, _ := newSvc()
	// recipient 空
	_, err := s.Create(context.Background(), 1, CreateInput{
		Recipient: "   ", Phone: "13800000000", Detail: "d",
	})
	require.Error(t, err)

	// detail 超长
	_, err = s.Create(context.Background(), 1, CreateInput{
		Recipient: "r", Phone: "13800000000", Detail: strings.Repeat("x", 201),
	})
	require.Error(t, err)

	// phone 过短
	_, err = s.Create(context.Background(), 1, CreateInput{
		Recipient: "r", Phone: "123", Detail: "d",
	})
	require.Error(t, err)
}

func TestService_Update_OK(t *testing.T) {
	s, _ := newSvc()
	a, _ := s.Create(context.Background(), 1, CreateInput{Recipient: "r", Phone: "13800000000", Detail: "d"})
	upd, err := s.Update(context.Background(), 1, a.ID, UpdateInput{
		Recipient: "new", Phone: "13800000001", Detail: "new",
	})
	require.NoError(t, err)
	assert.Equal(t, "new", upd.Recipient)
}

func TestService_Update_NotFound(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Update(context.Background(), 1, 999, UpdateInput{
		Recipient: "r", Phone: "13800000000", Detail: "d",
	})
	require.Error(t, err)
	var e *errs.Error
	require.True(t, errors.As(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

func TestService_SetDefault_NotFound(t *testing.T) {
	s, _ := newSvc()
	err := s.SetDefault(context.Background(), 1, 999)
	require.Error(t, err)
}

func TestService_Delete_OK(t *testing.T) {
	s, repo := newSvc()
	a, _ := s.Create(context.Background(), 1, CreateInput{Recipient: "r", Phone: "13800000000", Detail: "d"})
	require.NoError(t, s.Delete(context.Background(), 1, a.ID))
	n, _ := repo.CountByUser(context.Background(), 1)
	assert.Equal(t, 0, n)
}

func TestService_Delete_NotFound(t *testing.T) {
	s, _ := newSvc()
	err := s.Delete(context.Background(), 1, 999)
	require.Error(t, err)
}

// ---------- handler tests (httptest) ----------

// buildRouter 构造带 fake auth 的 gin engine。
func buildRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) {
		c.Set("user_user_id", int64(7))
		c.Set("user_role", "patient")
		c.Next()
	})
	h := NewHandler(svc)
	h.RegisterRoutes(g)
	return r
}

func do(t *testing.T, r *gin.Engine, method, path, body string) (int, httpx.Resp[map[string]any]) {
	t.Helper()
	var bodyR *strings.Reader
	if body != "" {
		bodyR = strings.NewReader(body)
	} else {
		bodyR = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyR)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func TestHandler_Create_201(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodPost, "/api/v1/addresses",
		`{"recipient":"张三","phone":"13800000000","detail":"中关村1号"}`)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Update_OK(t *testing.T) {
	s, _ := newSvc()
	s.Create(context.Background(), 7, CreateInput{Recipient: "张三", Phone: "13800000000", Detail: "a"})
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodPut, "/api/v1/addresses/1",
		`{"recipient":"李四","phone":"13900000000","detail":"b"}`)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Delete_OK(t *testing.T) {
	s, _ := newSvc()
	s.Create(context.Background(), 7, CreateInput{Recipient: "r", Phone: "13800000000", Detail: "d"})
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodDelete, "/api/v1/addresses/1", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_SetDefault_OK(t *testing.T) {
	s, _ := newSvc()
	s.Create(context.Background(), 7, CreateInput{Recipient: "r", Phone: "13800000000", Detail: "d"})
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodPut, "/api/v1/addresses/1/default", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_List_OK(t *testing.T) {
	s, _ := newSvc()
	s.Create(context.Background(), 7, CreateInput{Recipient: "r", Phone: "13800000000", Detail: "d"})
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodGet, "/api/v1/addresses", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_InvalidID(t *testing.T) {
	s, _ := newSvc()
	r := buildRouter(s)
	code, resp := do(t, r, http.MethodDelete, "/api/v1/addresses/abc", "")
	assert.Equal(t, http.StatusOK, code)
	assert.NotEqual(t, 0, resp.Code)
}