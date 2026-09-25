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

	"github.com/growdu/doctors/services/message/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// stubSvc 是 handler.Service 的最小 fake；测试中按需替换。
type stubSvc struct {
	sendMsg  *service.Message
	sendErr  error
	getMsg   *service.Message
	getErr   error
	listMsgs []*service.Message
	listErr  error
	bcastN   int
	bcastErr error
}

func (s *stubSvc) SendMessage(ctx context.Context, orderID, fromID, toID int64, body string) (*service.Message, error) {
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	return s.sendMsg, nil
}

func (s *stubSvc) GetByID(ctx context.Context, id int64) (*service.Message, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getMsg, nil
}

func (s *stubSvc) ListByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*service.Message, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listMsgs, nil
}

func (s *stubSvc) Broadcast(ctx context.Context, fromID int64, toIDs []int64, body string) (int, error) {
	if s.bcastErr != nil {
		return 0, s.bcastErr
	}
	return s.bcastN, nil
}

// 设置路由（手动注入 uid 到 ctx，避开 JWT 中间件）
func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	// 注入 uid
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

// TestSend_OK 验证 POST /api/v1/messages 走通。
func TestSend_OK(t *testing.T) {
	stub := &stubSvc{sendMsg: &service.Message{ID: 7, OrderID: 100, FromUserID: 42, ToUserID: 99, Body: "你好"}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/messages",
		`{"order_id":100,"to_user_id":99,"body":"你好"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, resp.Message)
	assert.Equal(t, int64(7), int64(resp.Data["id"].(float64)))
}

// TestSend_BadJSON 验证 body 解析失败 → 10001。
func TestSend_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/messages",
		`{"order_id":0}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestSend_ServiceError 验证 service 错误 → 业务码。
func TestSend_ServiceError(t *testing.T) {
	stub := &stubSvc{sendErr: errs.New(errs.CodeConflict, "dup")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/messages",
		`{"order_id":100,"to_user_id":99,"body":"x"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeConflict), resp.Code)
}

// TestSend_InternalError 验证非 errs.Error 错误 → 500000。
func TestSend_InternalError(t *testing.T) {
	stub := &stubSvc{sendErr: errors.New("boom")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/messages",
		`{"order_id":100,"to_user_id":99,"body":"x"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code)
}

// TestDetail_OK 验证 GET /api/v1/messages/:id。
func TestDetail_OK(t *testing.T) {
	stub := &stubSvc{getMsg: &service.Message{ID: 7, Body: "hi"}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/messages/7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestDetail_InvalidID 验证非法 id → 10001。
func TestDetail_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodGet, "/api/v1/messages/abc", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestDetail_NotFound 验证 service 返 not found → 12001。
func TestDetail_NotFound(t *testing.T) {
	stub := &stubSvc{getErr: errs.New(errs.CodeNotFound, "missing")}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/messages/99", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

// TestList_OK 验证 GET /api/v1/messages 走通。
func TestList_OK(t *testing.T) {
	stub := &stubSvc{listMsgs: []*service.Message{{ID: 1}, {ID: 2}}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/messages?order_id=100", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestBroadcast_OK 验证 POST /api/v1/messages/broadcast。
func TestBroadcast_OK(t *testing.T) {
	stub := &stubSvc{bcastN: 3}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/messages/broadcast",
		`{"to_user_ids":[2,3,4],"body":"系统通知"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(3), int64(resp.Data["sent"].(float64)))
}

// TestBroadcast_ParamInvalid 验证 body 缺字段 → 10001。
func TestBroadcast_ParamInvalid(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/messages/broadcast",
		`{"body":"x"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestBroadcast_ServiceError 验证 service 错误 → 业务码。
func TestBroadcast_ServiceError(t *testing.T) {
	stub := &stubSvc{bcastErr: errs.New(errs.CodeParamInvalid, "to_ids must be 1..500")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/messages/broadcast",
		`{"to_user_ids":[2,3,4],"body":"系统通知"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}
