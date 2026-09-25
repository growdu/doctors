package clients

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOrderServer 是 order-service 的 mock。
func fakeOrderServer(forceCancelCalled *int32, forceCancelID *int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/orders":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "data": []map[string]any{{"id": 1, "status": "paid"}},
			})
		case "/internal/orders/7/force-cancel":
			atomic.AddInt32(forceCancelCalled, 1)
			*forceCancelID = 7
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestOrderClient_ListAll_OK 验证调 order 内部接口拿全量订单。
func TestOrderClient_ListAll_OK(t *testing.T) {
	var called int32
	var gotID int64
	srv := fakeOrderServer(&called, &gotID)
	defer srv.Close()

	c := NewOrderClient(srv.URL, 5*time.Second)
	got, err := c.ListAll(context.Background(), ListOrdersParams{})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got, 1)
	assert.Equal(t, float64(1), got[0]["id"])
}

// TestOrderClient_ForceCancel_OK 验证 admin 调 order 强码。
func TestOrderClient_ForceCancel_OK(t *testing.T) {
	var called int32
	var gotID int64
	srv := fakeOrderServer(&called, &gotID)
	defer srv.Close()

	c := NewOrderClient(srv.URL, 5*time.Second)
	err := c.ForceCancel(context.Background(), 7, 1, "admin force")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
	assert.Equal(t, int64(7), gotID)
}

// TestOrderClient_ForceCancel_ServerError 验证 order 返 5xx → 转 ErrUpstreamUnavailable。
func TestOrderClient_ForceCancel_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewOrderClient(srv.URL, 5*time.Second)
	err := c.ForceCancel(context.Background(), 7, 1, "x")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrUpstreamUnavailable))
}

// TestOrderClient_ForceCancel_UpstreamBizErr 验证上游 body.code != 0 → 透传。
func TestOrderClient_ForceCancel_UpstreamBizErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 12002, "message": "order already canceled",
		})
	}))
	defer srv.Close()

	c := NewOrderClient(srv.URL, 5*time.Second)
	err := c.ForceCancel(context.Background(), 7, 1, "x")
	require.Error(t, err)
}

// fakeRefundServer refund-service mock。
func fakeRefundServer(approveCalled *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/internal/refunds" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "data": []map[string]any{{"id": 1, "amount": 100.0}},
			})
		case r.URL.Path == "/internal/refunds/1/approve" && r.Method == http.MethodPost:
			atomic.AddInt32(approveCalled, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestRefundClient_Approve_OK 验证 approve。
func TestRefundClient_Approve_OK(t *testing.T) {
	var called int32
	srv := fakeRefundServer(&called)
	defer srv.Close()

	c := NewRefundClient(srv.URL, 5*time.Second)
	err := c.Approve(context.Background(), 1, 1, "ok")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

// TestRefundClient_Reject_OK 验证 reject。
func TestRefundClient_Reject_OK(t *testing.T) {
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/refunds/1/reject" && r.Method == http.MethodPost {
			atomic.AddInt32(&called, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewRefundClient(srv.URL, 5*time.Second)
	err := c.Reject(context.Background(), 1, 1, "reject")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

// fakeEscortServer escort-service mock。
func fakeEscortServer(approveCalled, rejectCalled *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/internal/escorts/pending-audit":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "data": []map[string]any{{"id": 5, "nickname": "e"}},
			})
		case r.URL.Path == "/internal/escorts/5/approve":
			atomic.AddInt32(approveCalled, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
		case r.URL.Path == "/internal/escorts/5/reject":
			atomic.AddInt32(rejectCalled, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestEscortClient_PendingAudit 验证拿待审核。
func TestEscortClient_PendingAudit(t *testing.T) {
	var a, r int32
	srv := fakeEscortServer(&a, &r)
	defer srv.Close()

	c := NewEscortClient(srv.URL, 5*time.Second)
	got, err := c.ListPendingAudit(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

// TestEscortClient_Approve_Reject 验证通过/拒绝。
func TestEscortClient_Approve_Reject(t *testing.T) {
	var a, r int32
	srv := fakeEscortServer(&a, &r)
	defer srv.Close()

	c := NewEscortClient(srv.URL, 5*time.Second)
	require.NoError(t, c.Approve(context.Background(), 5, 1, "ok"))
	require.NoError(t, c.Reject(context.Background(), 5, 1, "no"))
	assert.Equal(t, int32(1), atomic.LoadInt32(&a))
	assert.Equal(t, int32(1), atomic.LoadInt32(&r))
}

// TestUserClient_List 验证 user client 调内部接口。
func TestUserClient_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/users" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "data": []map[string]any{{"id": 1, "role": "patient"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewUserClient(srv.URL, 5*time.Second)
	got, err := c.List(context.Background(), "patient", "", 1, 20)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}