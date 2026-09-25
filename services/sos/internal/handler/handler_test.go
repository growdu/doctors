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

	"github.com/growdu/doctors/services/sos/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct {
	raiseSOS  *service.SOS
	raiseErr  error
	getSOS    *service.SOS
	getErr    error
	listSOS   []*service.SOS
	listErr   error
	resolveErr error
}

func (s *stubSvc) Raise(ctx context.Context, orderID, userID int64, lat, lng float64, note string) (*service.SOS, error) {
	if s.raiseErr != nil {
		return nil, s.raiseErr
	}
	return s.raiseSOS, nil
}

func (s *stubSvc) GetByID(ctx context.Context, id int64) (*service.SOS, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getSOS, nil
}

func (s *stubSvc) List(ctx context.Context, f service.ListFilter) ([]*service.SOS, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listSOS, nil
}

func (s *stubSvc) Resolve(ctx context.Context, sosID int64) error {
	return s.resolveErr
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

// TestRaise_OK 验证 POST /api/v1/sos 走通。
func TestRaise_OK(t *testing.T) {
	stub := &stubSvc{raiseSOS: &service.SOS{ID: 7, OrderID: 100, UserID: 42, Status: "raised"}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/sos",
		`{"order_id":100,"lat":39.9,"lng":116.4,"note":"紧急"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.Equal(t, int64(7), int64(resp.Data["id"].(float64)))
}

// TestRaise_BadJSON 验证请求体缺字段 → 10001。
func TestRaise_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/sos", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestRaise_OrderNotActive 验证订单不活动 → 12002。
func TestRaise_OrderNotActive(t *testing.T) {
	stub := &stubSvc{raiseErr: errs.New(errs.CodeConflict, "order not active")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/sos",
		`{"order_id":100,"lat":39.9,"lng":116.4}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeConflict), resp.Code)
}

// TestRaise_InternalError 验证非 errs.Error → 500000。
func TestRaise_InternalError(t *testing.T) {
	stub := &stubSvc{raiseErr: errors.New("db down")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/sos",
		`{"order_id":100,"lat":39.9,"lng":116.4}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code)
}

// TestList_OK 验证 GET /api/v1/sos。
func TestList_OK(t *testing.T) {
	stub := &stubSvc{listSOS: []*service.SOS{{ID: 1}}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/sos?order_id=100", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestDetail_OK 验证 GET /api/v1/sos/:id。
func TestDetail_OK(t *testing.T) {
	stub := &stubSvc{getSOS: &service.SOS{ID: 7, Status: "raised"}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/sos/7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestDetail_InvalidID 验证非法 id → 10001。
func TestDetail_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodGet, "/api/v1/sos/abc", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestDetail_NotFound 验证 service 返 not found → 12001。
func TestDetail_NotFound(t *testing.T) {
	stub := &stubSvc{getErr: errs.New(errs.CodeNotFound, "sos not found")}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/sos/99", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

// TestResolve_OK 验证 POST /api/v1/sos/:id/resolve。
func TestResolve_OK(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/sos/7/resolve", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "resolved", resp.Data["status"])
}

// TestResolve_AlreadyResolved 验证重复 resolve → 12002。
func TestResolve_AlreadyResolved(t *testing.T) {
	stub := &stubSvc{resolveErr: errs.New(errs.CodeConflict, "already resolved")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/sos/7/resolve", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeConflict), resp.Code)
}

// TestResolve_InvalidID 验证非法 id → 10001。
func TestResolve_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/sos/abc/resolve", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}
