package refund

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeRepo 提供 PolicyRepo 的 fake 实现。
type fakeRepo struct {
	policies map[string]*Policy
}

func (f *fakeRepo) GetByScopeAndPhase(_ context.Context, scope string, phase Phase) (*Policy, error) {
	p, ok := f.policies[string(scope)+":"+string(phase)]
	if !ok {
		return nil, ErrInvalidPhase
	}
	return p, nil
}

func newRepo() *fakeRepo {
	return &fakeRepo{policies: map[string]*Policy{
		"default:before_paid":     {PhaseBeforePaid, 100, 0},
		"default:after_paid_5min": {PhaseAfterPaid5Min, 100, 0},
		"default:after_accepted":  {PhaseAfterAccepted, 95, 5},
		"default:in_service":      {PhaseInService, 0, 0},
	}}
}

func ptrTime(t time.Time) *time.Time { return &t }

// TestDetectPhase 表驱动覆盖各阶段。
func TestDetectPhase(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		ctx  OrderContext
		want Phase
	}{
		{"未付款", OrderContext{Now: now}, PhaseBeforePaid},
		{"付款后 1 分钟（未接单）", OrderContext{Now: now, PaidAt: ptrTime(now.Add(-1 * time.Minute))}, PhaseAfterPaid5Min},
		{"付款后 6 分钟（未接单）", OrderContext{Now: now, PaidAt: ptrTime(now.Add(-6 * time.Minute))}, PhaseAfterPaid5Min},
		{"接单后服务未开始", OrderContext{
			Now: now, PaidAt: ptrTime(now.Add(-1 * time.Hour)),
			AcceptedAt: ptrTime(now.Add(-30 * time.Minute)),
			ServiceStartAt: now.Add(time.Hour),
		}, PhaseAfterAccepted},
		{"服务已开始", OrderContext{
			Now: now, PaidAt: ptrTime(now.Add(-2 * time.Hour)),
			AcceptedAt: ptrTime(now.Add(-1 * time.Hour)),
			ServiceStartAt: now.Add(-10 * time.Minute),
		}, PhaseInService},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, DetectPhase(c.ctx))
		})
	}
}

// TestDecide_RefundPercent 表驱动覆盖退款比例计算。
func TestDecide_RefundPercent(t *testing.T) {
	now := time.Now()
	paid := ptrTime(now.Add(-1 * time.Hour))
	accepted := ptrTime(now.Add(-30 * time.Minute))
	serviceStart := now.Add(time.Hour)

	cases := []struct {
		name    string
		ctx     OrderContext
		wantOk  bool
		wantPct float64
	}{
		{"未付款 100%", OrderContext{Now: now}, true, 100},
		{"付款 1 分钟 100%", OrderContext{Now: now, PaidAt: paid, ServiceStartAt: serviceStart}, true, 100},
		{"付款 10 分钟 100%", OrderContext{Now: now, PaidAt: ptrTime(now.Add(-10 * time.Minute)), ServiceStartAt: serviceStart}, true, 100},
		{"接单后 95%", OrderContext{Now: now, PaidAt: paid, AcceptedAt: accepted, ServiceStartAt: serviceStart}, true, 95},
		{"服务已开始 0%", OrderContext{Now: now, PaidAt: paid, AcceptedAt: accepted, ServiceStartAt: now.Add(-5 * time.Minute)}, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := Decide(c.ctx, newRepo(), "default")
			assert.Equal(t, c.wantOk, d.Eligible)
			assert.InDelta(t, c.wantPct, d.RefundPercent, 0.01)
		})
	}
}

// TestRefundAmount 验证金额计算。
func TestRefundAmount(t *testing.T) {
	ctx := OrderContext{Amount: 200.00, Now: time.Now()}
	d := Decision{Eligible: true, RefundPercent: 95, EscortCompensation: 5}
	refund, escort := RefundAmount(ctx, d)
	assert.InDelta(t, 190.0, refund, 0.01)
	assert.InDelta(t, 10.0, escort, 0.01)
}

// TestRefundAmount_NotEligible 验证 ineligible 时金额为 0。
func TestRefundAmount_NotEligible(t *testing.T) {
	ctx := OrderContext{Amount: 200, Now: time.Now()}
	refund, escort := RefundAmount(ctx, Decision{Eligible: false})
	assert.Equal(t, 0.0, refund)
	assert.Equal(t, 0.0, escort)
}

// TestDecide_NoPolicy 验证没匹配 policy 时返回 not eligible。
func TestDecide_NoPolicy(t *testing.T) {
	now := time.Now()
	ctx := OrderContext{Now: now, PaidAt: ptrTime(now.Add(-1 * time.Hour)),
		AcceptedAt: ptrTime(now.Add(-30 * time.Minute)), ServiceStartAt: now.Add(time.Hour)}
	empty := &fakeRepo{policies: map[string]*Policy{}}
	d := Decide(ctx, empty, "default")
	assert.False(t, d.Eligible)
	assert.Equal(t, "no_policy_for_phase", d.Reason)
}