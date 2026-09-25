package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/review/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct {
	createRv  *service.Review
	createErr error
	getRv     *service.Review
	getErr    error
	listRvs   []*service.Review
	listErr   error
	replyRv   *service.Review
	replyErr  error
}

func (s *stubSvc) CreateReview(ctx context.Context, reviewerID, orderID, escortID int64, rating int, comment string) (*service.Review, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createRv, nil
}

func (s *stubSvc) GetByID(ctx context.Context, id int64) (*service.Review, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getRv, nil
}

func (s *stubSvc) List(ctx context.Context, f service.ListFilter) ([]*service.Review, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listRvs, nil
}

func (s *stubSvc) Reply(ctx context.Context, id, adminID int64, body string) (*service.Review, error) {
	if s.replyErr != nil {
		return nil, s.replyErr
	}
	return s.replyRv, nil
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set("uid", int64(42))
		c.Next()
	})
	h.RegisterRoutes(v1)
	return r
}

func doJSON(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestCreate_OK 验证 POST /api/v1/reviews 走通。
func TestCreate_OK(t *testing.T) {
	stub := &stubSvc{createRv: &service.Review{ID: 7, OrderID: 100, EscortID: 99, Rating: 5}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/reviews",
		`{"order_id":100,"escort_id":99,"rating":5,"comment":"好"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.Equal(t, int64(7), int64(resp.Data["id"].(float64)))
}

// TestCreate_BadJSON 验证 body 缺字段 → 10001。
func TestCreate_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/reviews", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestCreate_Duplicate 验证 service 返 conflict → 12002。
func TestCreate_Duplicate(t *testing.T) {
	stub := &stubSvc{createErr: errs.New(errs.CodeConflict, "dup")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/reviews",
		`{"order_id":100,"escort_id":99,"rating":5}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeConflict), resp.Code)
}

// TestCreate_InternalError 验证非 errs.Error → 500000。
func TestCreate_InternalError(t *testing.T) {
	stub := &stubSvc{createErr: errors.New("db down")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/reviews",
		`{"order_id":100,"escort_id":99,"rating":5}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code)
}

// TestList_OK 验证 GET /api/v1/reviews。
func TestList_OK(t *testing.T) {
	stub := &stubSvc{listRvs: []*service.Review{{ID: 1}}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/reviews?escort_id=7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestDetail_OK 验证 GET /api/v1/reviews/:id。
func TestDetail_OK(t *testing.T) {
	stub := &stubSvc{getRv: &service.Review{ID: 7, Rating: 5}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/reviews/7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestDetail_InvalidID 验证非法 id → 10001。
func TestDetail_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodGet, "/api/v1/reviews/abc", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestDetail_NotFound 验证 service 返 not found → 12001。
func TestDetail_NotFound(t *testing.T) {
	stub := &stubSvc{getErr: errs.New(errs.CodeNotFound, "missing")}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/reviews/99", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

// TestReply_OK 验证 POST /api/v1/reviews/:id/reply。
func TestReply_OK(t *testing.T) {
	stub := &stubSvc{replyRv: &service.Review{ID: 7, Reply: "感谢反馈", RepliedBy: 42}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/reviews/7/reply",
		`{"body":"感谢反馈"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestReply_BadJSON 验证 body 缺字段 → 10001。
func TestReply_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/reviews/7/reply", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestReply_Conflict 验证已回复 → 12002。
func TestReply_Conflict(t *testing.T) {
	stub := &stubSvc{replyErr: errs.New(errs.CodeConflict, "already replied")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/reviews/7/reply",
		`{"body":"再次"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeConflict), resp.Code)
}

// TestReply_InvalidID 验证非法 id → 10001。
func TestReply_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/reviews/abc/reply",
		`{"body":"x"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}
