package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScannerRepo 实现 RepoScanner 接口（最小子集）。
type fakeScannerRepo struct {
	mu          sync.Mutex
	completed   []CompletedOrder
	unfreezeErr error
	listErr     error
	calls       int
}

func newFakeScannerRepo() *fakeScannerRepo {
	return &fakeScannerRepo{}
}

func (f *fakeScannerRepo) ListCompletedOrdersBefore(ctx context.Context, cutoff time.Time, limit int) ([]CompletedOrder, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]CompletedOrder, 0, len(f.completed))
	for _, o := range f.completed {
		out = append(out, o)
	}
	return out, nil
}
func (f *fakeScannerRepo) UnfreezeToBalance(ctx context.Context, uid, orderID int64, amount decimal.Decimal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++ // count attempts
	if f.unfreezeErr != nil {
		return f.unfreezeErr
	}
	return nil
}

// 编译期检查 fakeScannerRepo 满足 RepoScanner 接口。
var _ RepoScanner = (*fakeScannerRepo)(nil)

// TestScanner_ReleasesExpiredOrder 验证已超期的订单触发 UnfreezeToBalance。
func TestScanner_ReleasesExpiredOrder(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()
	r.completed = []CompletedOrder{
		{OrderID: 42, UserID: 100, Amount: decimal.NewFromInt(300)},
	}

	s := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	require.NoError(t, s.RunScanOnce(context.Background(), now))
	assert.Equal(t, 1, r.calls)
}

// TestScanner_NoReleaseBeforeThreshold 验证无订单时不释放。
func TestScanner_NoReleaseBeforeThreshold(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()

	s := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	require.NoError(t, s.RunScanOnce(context.Background(), now))
	assert.Equal(t, 0, r.calls)
}

// TestScanner_ThresholdParametrized 验证 1 分钟 vs 7 天阈值都用同一段代码。
func TestScanner_ThresholdParametrized(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()
	r.completed = []CompletedOrder{
		{OrderID: 42, UserID: 100, Amount: decimal.NewFromInt(300)},
	}

	// 1 分钟阈值
	s1 := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	require.NoError(t, s1.RunScanOnce(context.Background(), now))
	assert.Equal(t, 1, r.calls)

	// 7 天阈值
	s2 := NewScanner(r, 7*24*time.Hour, func() time.Time { return now })
	r.calls = 0
	require.NoError(t, s2.RunScanOnce(context.Background(), now))
	assert.Equal(t, 1, r.calls, "不同阈值，同一段代码触发 1 次")
}

// TestScanner_PropagatesListError 验证 repo list 错误不 panic。
func TestScanner_PropagatesListError(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()
	r.listErr = assert.AnError

	s := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	err := s.RunScanOnce(context.Background(), now)
	require.Error(t, err)
}

// TestScanner_PropagatesUnfreezeError 验证单笔 unfreeze 失败不影响其他单。
func TestScanner_PropagatesUnfreezeError(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()
	r.completed = []CompletedOrder{
		{OrderID: 1, UserID: 100, Amount: decimal.NewFromInt(100)},
		{OrderID: 2, UserID: 100, Amount: decimal.NewFromInt(200)},
	}
	r.unfreezeErr = assert.AnError

	s := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	// 单笔失败 → RunScanOnce 不中断（warn log + continue），但实际调用次数反映 fail。
	// 由于 fake 一旦设置 unfreezeErr 所有调用都失败，calls 应等于 completed len。
	err := s.RunScanOnce(context.Background(), now)
	require.NoError(t, err, "单笔失败不返回 err（warn log 内部）")
	assert.Equal(t, 2, r.calls, "即使报错也调用了 2 次（best-effort）")
}

// TestScanner_SkipZeroAmount 验证 amount <= 0 跳过。
func TestScanner_SkipZeroAmount(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	r := newFakeScannerRepo()
	r.completed = []CompletedOrder{
		{OrderID: 1, UserID: 100, Amount: decimal.Zero},
		{OrderID: 2, UserID: 100, Amount: decimal.NewFromInt(-50)},
	}

	s := NewScanner(r, 1*time.Minute, func() time.Time { return now })
	require.NoError(t, s.RunScanOnce(context.Background(), now))
	assert.Equal(t, 0, r.calls, "0/负金额跳过")
}