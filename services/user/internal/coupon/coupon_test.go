package coupon

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
	mu       sync.Mutex
	coupons  map[int64]*Record
	uc       map[int64]*UserCouponWithTemplate
	nextCID  int64
	nextUCID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		coupons:  map[int64]*Record{},
		uc:       map[int64]*UserCouponWithTemplate{},
		nextCID:  1,
		nextUCID: 1,
	}
}

func (r *fakeRepo) ListActive(ctx context.Context, limit, offset int) ([]*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*Record{}
	for _, c := range r.coupons {
		if c.Status == "active" {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.coupons[id]
	if !ok {
		return nil, ErrCouponNotFound
	}
	return c, nil
}

func (r *fakeRepo) Claim(ctx context.Context, userID, couponID int64) (*UserCoupon, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.coupons[couponID]
	if !ok {
		return nil, ErrCouponNotFound
	}
	if c.Stock <= 0 || c.Status != "active" {
		return nil, ErrNoStock
	}
	// 检查是否已领取
	for _, x := range r.uc {
		if x.UserID == userID && x.CouponID == couponID {
			return nil, ErrAlreadyClaimed
		}
	}
	c.Stock--
	uc := &UserCoupon{
		ID:        r.nextUCID,
		UserID:    userID,
		CouponID:  couponID,
		Status:    "unused",
		ExpiresAt: c.ValidUntil,
		ClaimedAt: time.Now(),
	}
	r.nextUCID++
	r.uc[uc.ID] = &UserCouponWithTemplate{UserCoupon: *uc, Coupon: *c}
	return uc, nil
}

func (r *fakeRepo) ListByUser(ctx context.Context, userID int64) ([]*UserCouponWithTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*UserCouponWithTemplate{}
	for _, x := range r.uc {
		if x.UserID == userID {
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *fakeRepo) MarkUsed(ctx context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.uc[id]
	if !ok || x.UserID != userID {
		return ErrUserCouponNotFound
	}
	if x.Status == "used" {
		return ErrAlreadyClaimed
	}
	if time.Now().After(x.ExpiresAt) {
		x.Status = "expired"
		return ErrUserCouponNotFound
	}
	now := time.Now()
	x.Status = "used"
	x.UsedAt = &now
	return nil
}

// ---------- service tests ----------

func newSvc() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo), repo
}

func seedCoupon(s *Service, repo *fakeRepo, name string, validHours int) *Record {
	c := &Record{
		ID:         repo.nextCID,
		Name:       name,
		Type:       "amount_off",
		Value:      20,
		Threshold:  100,
		ValidFrom:  time.Now().Add(-time.Hour),
		ValidUntil: time.Now().Add(time.Duration(validHours) * time.Hour),
		Stock:      100,
		Status:     "active",
	}
	repo.nextCID++
	repo.coupons[c.ID] = c
	return c
}

func TestListActive_OK(t *testing.T) {
	s, repo := newSvc()
	seedCoupon(s, repo, "满 100 减 20", 720)
	list, err := s.ListActive(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestListActive_Empty(t *testing.T) {
	s, _ := newSvc()
	list, err := s.ListActive(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestGet_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	got, err := s.Get(context.Background(), c.ID)
	require.NoError(t, err)
	assert.Equal(t, "x", got.Name)
}

func TestGet_NotFound(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Get(context.Background(), 999)
	var e *errs.Error
	require.True(t, asErr(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

func TestClaim_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	uc, err := s.Claim(context.Background(), 1, c.ID)
	require.NoError(t, err)
	require.NotNil(t, uc)
	assert.NotZero(t, uc.ID)
	assert.Equal(t, "unused", uc.Status)
}

func TestClaim_AlreadyClaimed(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	_, err := s.Claim(context.Background(), 1, c.ID)
	require.NoError(t, err)
	_, err = s.Claim(context.Background(), 1, c.ID)
	var e *errs.Error
	require.True(t, asErr(err, &e))
	assert.Equal(t, errs.CodeConflict, e.Code)
}

func TestClaim_NotFound(t *testing.T) {
	s, _ := newSvc()
	_, err := s.Claim(context.Background(), 1, 999)
	var e *errs.Error
	require.True(t, asErr(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

func TestListMine_OK(t *testing.T) {
	s, repo := newSvc()
	c1 := seedCoupon(s, repo, "a", 720)
	c2 := seedCoupon(s, repo, "b", 720)
	s.Claim(context.Background(), 1, c1.ID)
	s.Claim(context.Background(), 1, c2.ID)
	list, err := s.ListMine(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUse_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	uc, _ := s.Claim(context.Background(), 1, c.ID)
	require.NoError(t, s.Use(context.Background(), 1, uc.ID))
}

func TestUse_AlreadyUsed(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	uc, _ := s.Claim(context.Background(), 1, c.ID)
	require.NoError(t, s.Use(context.Background(), 1, uc.ID))
	err := s.Use(context.Background(), 1, uc.ID)
	var e *errs.Error
	require.True(t, asErr(err, &e))
	assert.Equal(t, errs.CodeConflict, e.Code)
}

func TestUse_NotFound(t *testing.T) {
	s, _ := newSvc()
	err := s.Use(context.Background(), 1, 999)
	var e *errs.Error
	require.True(t, asErr(err, &e))
	assert.Equal(t, errs.CodeNotFound, e.Code)
}

// ---------- handler tests ----------

func buildRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(func(c *gin.Context) {
		c.Set("user_user_id", int64(7))
		c.Next()
	})
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

func TestHandler_List(t *testing.T) {
	s, repo := newSvc()
	seedCoupon(s, repo, "x", 720)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/coupons", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Get_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/coupons/"+itoa(c.ID), "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Claim_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodPost, "/api/v1/coupons/"+itoa(c.ID)+"/claim", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_ListMine(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	s.Claim(context.Background(), 7, c.ID)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodGet, "/api/v1/me/coupons", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

func TestHandler_Use_OK(t *testing.T) {
	s, repo := newSvc()
	c := seedCoupon(s, repo, "x", 720)
	uc, _ := s.Claim(context.Background(), 7, c.ID)
	r := buildRouter(s)
	code, resp := doReq(t, r, http.MethodPost, "/api/v1/me/coupons/"+itoa(uc.ID)+"/use", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 0, resp.Code)
}

// ---------- helpers ----------

func asErr(err error, e **errs.Error) bool {
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