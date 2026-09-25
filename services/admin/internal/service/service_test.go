package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/admin/internal/clients"
	"github.com/growdu/doctors/shared/errs"
)

// ---------- fake clients ----------

type fakeOrderClient struct {
	forceCancelCalled int32
	forceCancelID     int64
	forceCancelErr    error
	listOut           []map[string]any
	listErr           error
}

func (f *fakeOrderClient) ListAll(_ context.Context, _ clients.ListOrdersParams) ([]map[string]any, error) {
	return f.listOut, f.listErr
}
func (f *fakeOrderClient) ForceCancel(_ context.Context, orderID, _ int64, _ string) error {
	atomic.AddInt32(&f.forceCancelCalled, 1)
	f.forceCancelID = orderID
	return f.forceCancelErr
}

type fakeRefundClient struct {
	approveCalled int32
	rejectCalled  int32
	approveErr    error
	rejectErr     error
}

func (f *fakeRefundClient) List(_ context.Context, _ string, _, _ int) ([]map[string]any, error) {
	return []map[string]any{{"id": 1}}, nil
}
func (f *fakeRefundClient) Approve(_ context.Context, _, _ int64, _ string) error {
	atomic.AddInt32(&f.approveCalled, 1)
	return f.approveErr
}
func (f *fakeRefundClient) Reject(_ context.Context, _, _ int64, _ string) error {
	atomic.AddInt32(&f.rejectCalled, 1)
	return f.rejectErr
}

type fakeEscortClient struct {
	approveCalled int32
	rejectCalled  int32
	approveErr    error
	rejectErr     error
}

func (f *fakeEscortClient) ListPendingAudit(_ context.Context) ([]map[string]any, error) {
	return []map[string]any{{"id": 5}}, nil
}
func (f *fakeEscortClient) Approve(_ context.Context, _, _ int64, _ string) error {
	atomic.AddInt32(&f.approveCalled, 1)
	return f.approveErr
}
func (f *fakeEscortClient) Reject(_ context.Context, _, _ int64, _ string) error {
	atomic.AddInt32(&f.rejectCalled, 1)
	return f.rejectErr
}

type fakeUserClient struct{ listOut []map[string]any }

func (f *fakeUserClient) List(_ context.Context, _, _ string, _, _ int) ([]map[string]any, error) {
	return f.listOut, nil
}

type fakePublisher struct {
	events int32
}

func (f *fakePublisher) Publish(_ context.Context, _ string, _ any) error {
	atomic.AddInt32(&f.events, 1)
	return nil
}

type fakeReportsRepo struct {
	out *OverviewStats
}

func (f *fakeReportsRepo) Overview(_ context.Context) (*OverviewStats, error) {
	return f.out, nil
}

// TestOrders_ForceCancel 验证 service.ForceCancelOrder 调 order client + 发事件。
func TestOrders_ForceCancel(t *testing.T) {
	oc := &fakeOrderClient{}
	pub := &fakePublisher{}
	s := New(nil, nil, oc, nil, nil, nil, pub)
	err := s.ForceCancelOrder(context.Background(), 7, 1, "admin")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&oc.forceCancelCalled))
	assert.Equal(t, int64(7), oc.forceCancelID)
	assert.Equal(t, int32(1), pub.events) // 发 AdminOrderForceCancelledEvent
}

// TestOrders_ForceCancel_UpstreamErr 验证上游错误透传（不重发事件）。
func TestOrders_ForceCancel_UpstreamErr(t *testing.T) {
	oc := &fakeOrderClient{forceCancelErr: errs.New(errs.CodeUpstreamUnavailable, "down")}
	pub := &fakePublisher{}
	s := New(nil, nil, oc, nil, nil, nil, pub)
	err := s.ForceCancelOrder(context.Background(), 7, 1, "admin")
	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&oc.forceCancelCalled))
	assert.Equal(t, int32(0), pub.events, "失败不应发事件")
}

// TestReports_Overview_Cache 验证 30s 缓存命中。
func TestReports_Overview_Cache(t *testing.T) {
	rr := &fakeReportsRepo{out: &OverviewStats{TodayOrders: 1}}
	s := New(rr, nil, nil, nil, nil, nil, nil)

	got1, err := s.OverviewStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, got1.TodayOrders)

	// 改 source → 第二次应仍返回缓存
	rr.out = &OverviewStats{TodayOrders: 99}
	got2, err := s.OverviewStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, got2.TodayOrders, "30s 内应仍命中缓存")

	// 强制过期 → 返回新值
	s.ForceExpire()
	got3, _ := s.OverviewStats(context.Background())
	assert.Equal(t, 99, got3.TodayOrders)
}

// TestRefunds_ApproveReject 验证 service 调 refund client + 发事件。
func TestRefunds_ApproveReject(t *testing.T) {
	rc := &fakeRefundClient{}
	pub := &fakePublisher{}
	s := New(nil, nil, nil, rc, nil, nil, pub)

	require.NoError(t, s.ApproveRefund(context.Background(), 1, 100, 1, "ok"))
	require.NoError(t, s.RejectRefund(context.Background(), 2, 100, 1, "no"))
	assert.Equal(t, int32(1), atomic.LoadInt32(&rc.approveCalled))
	assert.Equal(t, int32(1), atomic.LoadInt32(&rc.rejectCalled))
	assert.Equal(t, int32(2), pub.events)
}

// TestEscorts_ApproveReject 验证陪诊师审核。
func TestEscorts_ApproveReject(t *testing.T) {
	ec := &fakeEscortClient{}
	pub := &fakePublisher{}
	s := New(nil, nil, nil, nil, ec, nil, pub)

	require.NoError(t, s.ApproveEscort(context.Background(), 5, 1, "ok"))
	require.NoError(t, s.RejectEscort(context.Background(), 5, 1, "no"))
	assert.Equal(t, int32(1), ec.approveCalled)
	assert.Equal(t, int32(1), ec.rejectCalled)
	assert.Equal(t, int32(2), pub.events)
}

// TestEscort_UpstreamErr 验证审核失败透传。
func TestEscort_UpstreamErr(t *testing.T) {
	ec := &fakeEscortClient{approveErr: errors.New("escort service down")}
	pub := &fakePublisher{}
	s := New(nil, nil, nil, nil, ec, nil, pub)

	err := s.ApproveEscort(context.Background(), 5, 1, "ok")
	require.Error(t, err)
	assert.Equal(t, int32(0), pub.events)
}

// TestUsers_List 验证 user client 透传。
func TestUsers_List(t *testing.T) {
	uc := &fakeUserClient{listOut: []map[string]any{{"id": 1}}}
	s := New(nil, nil, nil, nil, nil, uc, nil)
	got, err := s.ListUsers(context.Background(), "patient", "", 1, 10)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

// 辅助函数避免 unused import。
var _ = time.Now