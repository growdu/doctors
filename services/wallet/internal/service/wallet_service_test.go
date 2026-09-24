package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/shared/errs"
)

// fakeRepo 实现 Repo 接口（最小子集）。
type fakeRepo struct {
	wallets       map[int64]*repo.Wallet
	withdrawals   map[int64]*repo.Withdrawal
	billings      []*repo.Billing
	freezeCalls   int
	unfreezeCalls int
	deductCalls   int

	// 注入错误
	freezeErr   error
	unfreezeErr error
	deductErr   error
	withdrawErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		wallets:     map[int64]*repo.Wallet{},
		withdrawals: map[int64]*repo.Withdrawal{},
	}
}

func (f *fakeRepo) Get(ctx context.Context, uid int64) (*repo.Wallet, error) {
	return f.wallets[uid], nil
}
func (f *fakeRepo) GetOrCreate(ctx context.Context, uid int64) (*repo.Wallet, error) {
	if w, ok := f.wallets[uid]; ok {
		return w, nil
	}
	w := &repo.Wallet{UserID: uid, Currency: "CNY", CreatedAt: time.Now()}
	f.wallets[uid] = w
	return w, nil
}
func (f *fakeRepo) FreezeIncome(ctx context.Context, uid, orderID int64, amount decimal.Decimal) error {
	if f.freezeErr != nil {
		return f.freezeErr
	}
	w, _ := f.GetOrCreate(ctx, uid)
	w.Frozen = w.Frozen.Add(amount)
	w.TotalEarned = w.TotalEarned.Add(amount)
	f.freezeCalls++
	oid := orderID
	f.billings = append(f.billings, &repo.Billing{
		UserID: uid, OrderID: &oid, Type: "order_income",
		Amount: amount, BalanceAfter: w.Balance, FrozenAfter: w.Frozen,
	})
	return nil
}
func (f *fakeRepo) DeductFrozenForRefund(ctx context.Context, uid, orderID int64, amount decimal.Decimal) error {
	if f.deductErr != nil {
		return f.deductErr
	}
	w := f.wallets[uid]
	if w == nil {
		return errors.New("no wallet")
	}
	w.Frozen = w.Frozen.Sub(amount)
	w.TotalEarned = w.TotalEarned.Sub(amount)
	f.deductCalls++
	oid := orderID
	f.billings = append(f.billings, &repo.Billing{
		UserID: uid, OrderID: &oid, Type: "refund_deduct",
		Amount: amount.Neg(), BalanceAfter: w.Balance, FrozenAfter: w.Frozen,
	})
	return nil
}
func (f *fakeRepo) UnfreezeToBalance(ctx context.Context, uid, orderID int64, amount decimal.Decimal) error {
	if f.unfreezeErr != nil {
		return f.unfreezeErr
	}
	w, _ := f.GetOrCreate(ctx, uid)
	w.Frozen = w.Frozen.Sub(amount)
	w.Balance = w.Balance.Add(amount)
	f.unfreezeCalls++
	oid := orderID
	f.billings = append(f.billings, &repo.Billing{
		UserID: uid, OrderID: &oid, Type: "frozen_release",
		Amount: amount, BalanceAfter: w.Balance, FrozenAfter: w.Frozen,
	})
	return nil
}
func (f *fakeRepo) CreateWithdrawal(ctx context.Context, w *repo.Withdrawal) error {
	if f.withdrawErr != nil {
		return f.withdrawErr
	}
	wallet := f.wallets[w.UserID]
	if wallet == nil {
		return errors.New("no wallet")
	}
	if wallet.Balance.LessThan(w.Amount) {
		return repo.ErrInsufficientBalance
	}
	wallet.Balance = wallet.Balance.Sub(w.Amount)
	wallet.TotalWithdrawn = wallet.TotalWithdrawn.Add(w.Amount)
	w.ID = int64(len(f.withdrawals) + 1)
	w.Status = "pending"
	w.CreatedAt = time.Now()
	f.withdrawals[w.ID] = w
	f.billings = append(f.billings, &repo.Billing{
		UserID: w.UserID, Type: "withdraw",
		Amount: w.Amount.Neg(), BalanceAfter: wallet.Balance, FrozenAfter: wallet.Frozen,
	})
	return nil
}
func (f *fakeRepo) GetWithdrawal(ctx context.Context, id int64) (*repo.Withdrawal, error) {
	return f.withdrawals[id], nil
}
func (f *fakeRepo) MarkWithdrawalApproved(ctx context.Context, id, reviewerID int64) error {
	w := f.withdrawals[id]
	w.Status = "approved"
	now := time.Now()
	w.ReviewedAt = &now
	w.ReviewedBy = &reviewerID
	return nil
}
func (f *fakeRepo) MarkWithdrawalPaid(ctx context.Context, id int64, externalTxID string) error {
	w := f.withdrawals[id]
	w.Status = "paid"
	w.ExternalTxID = externalTxID
	now := time.Now()
	w.PaidAt = &now
	return nil
}
func (f *fakeRepo) RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error {
	w := f.withdrawals[id]
	w.Status = "rejected"
	now := time.Now()
	w.ReviewedAt = &now
	w.ReviewedBy = &reviewerID
	w.FailureReason = reason
	wallet := f.wallets[w.UserID]
	wallet.Balance = wallet.Balance.Add(w.Amount)
	wallet.TotalWithdrawn = wallet.TotalWithdrawn.Sub(w.Amount)
	return nil
}
func (f *fakeRepo) ListTransactions(ctx context.Context, uid int64, limit, offset int) ([]*repo.Billing, error) {
	out := []*repo.Billing{}
	for _, b := range f.billings {
		if b.UserID == uid {
			out = append(out, b)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListPendingWithdrawals(ctx context.Context, limit int) ([]*repo.Withdrawal, error) {
	out := []*repo.Withdrawal{}
	for _, w := range f.withdrawals {
		if w.Status == "pending" {
			out = append(out, w)
		}
	}
	return out, nil
}

// TestWalletService_GetWallet 验证 GET 返回 wallet；用户无钱包返回零值。
func TestWalletService_GetWallet(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	v, err := svc.GetWallet(context.Background(), 100)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, int64(100), v.UserID)
	assert.True(t, v.Balance.IsZero())
}

// TestWalletService_OnOrderCompleted 验证订单完成入 frozen。
func TestWalletService_OnOrderCompleted(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	err := svc.OnOrderCompleted(context.Background(), 100, 1, decimal.NewFromInt(300))
	require.NoError(t, err)
	w := r.wallets[100]
	assert.True(t, w.Frozen.Equal(decimal.NewFromInt(300)))
}

// TestWalletService_OnOrderCompleted_NoEscort 验证无 escort 跳过。
func TestWalletService_OnOrderCompleted_NoEscort(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	err := svc.OnOrderCompleted(context.Background(), 0, 1, decimal.NewFromInt(300))
	require.NoError(t, err, "escort_id=0 应直接跳过；不报错")
	assert.Empty(t, r.wallets, "无 escort 不创建 wallet")
}

// TestWalletService_OnOrderCompleted_ZeroAmount 验证金额为 0 也跳过。
func TestWalletService_OnOrderCompleted_ZeroAmount(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	err := svc.OnOrderCompleted(context.Background(), 100, 1, decimal.Zero)
	require.NoError(t, err)
	assert.Empty(t, r.wallets, "金额 0 不创建 wallet")
}

// TestWalletService_OnRefund 验证退款扣 frozen。
func TestWalletService_OnRefund(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, svc.OnOrderCompleted(context.Background(), 100, 1, decimal.NewFromInt(300)))
	require.NoError(t, svc.OnRefund(context.Background(), 100, 1, decimal.NewFromInt(150)))
	w := r.wallets[100]
	assert.True(t, w.Frozen.Equal(decimal.NewFromInt(150)))
}

// TestWalletService_CreateWithdrawal_OK 验证提现成功。
func TestWalletService_CreateWithdrawal_OK(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(500)))

	w, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(200), "wx", "138****0000")
	require.NoError(t, err)
	require.NotNil(t, w)
	assert.Equal(t, "pending", w.Status)
	assert.NotZero(t, w.ID)
	assert.True(t, r.wallets[100].Balance.Equal(decimal.NewFromInt(300)))
}

// TestWalletService_CreateWithdrawal_BelowMinimum 验证 < 100 元被拒。
func TestWalletService_CreateWithdrawal_BelowMinimum(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(500)))

	_, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(50), "wx", "138****0000")
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeParamInvalid, e.Code, "< 100 元应返回参数无效")
}

// TestWalletService_CreateWithdrawal_InsufficientBalance 验证余额不足。
func TestWalletService_CreateWithdrawal_InsufficientBalance(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(50)))

	_, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(100), "wx", "138****0000")
	require.Error(t, err)
}

// TestWalletService_CreateWithdrawal_InvalidChannel 验证非法 channel 被拒。
func TestWalletService_CreateWithdrawal_InvalidChannel(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(500)))

	_, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(200), "bitcoin", "x")
	require.Error(t, err)
	e, ok := errs.As(err)
	require.True(t, ok)
	assert.Equal(t, errs.CodeParamInvalid, e.Code)
}

// TestWalletService_ApproveAndPay 验证审核通过 + 打款完成（mock）。
func TestWalletService_ApproveAndPay(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(200)))
	w, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(200), "wx", "138****0000")
	require.NoError(t, err)

	require.NoError(t, svc.ApproveWithdrawal(context.Background(), w.ID, 999))
	require.NoError(t, svc.MarkWithdrawalPaid(context.Background(), w.ID))

	got, _ := r.GetWithdrawal(context.Background(), w.ID)
	require.NotNil(t, got)
	assert.Equal(t, "paid", got.Status)
	assert.Contains(t, got.ExternalTxID, "MOCK-TX-")
}

// TestWalletService_Reject 验证审核拒绝 + 余额回滚。
func TestWalletService_Reject(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, r.UnfreezeToBalance(context.Background(), 100, 0, decimal.NewFromInt(200)))
	w, err := svc.CreateWithdrawal(context.Background(), 100, decimal.NewFromInt(200), "wx", "138****0000")
	require.NoError(t, err)

	require.NoError(t, svc.RejectWithdrawal(context.Background(), w.ID, 999, "bank card invalid"))

	got, _ := r.GetWithdrawal(context.Background(), w.ID)
	require.NotNil(t, got)
	assert.Equal(t, "rejected", got.Status)
	assert.Equal(t, "bank card invalid", got.FailureReason)
	assert.True(t, r.wallets[100].Balance.Equal(decimal.NewFromInt(200)), "余额回滚 200")
}

// TestWalletService_ListTransactions 验证账单列表。
func TestWalletService_ListTransactions(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, svc.OnOrderCompleted(context.Background(), 100, 1, decimal.NewFromInt(300)))
	require.NoError(t, svc.OnOrderCompleted(context.Background(), 100, 2, decimal.NewFromInt(200)))

	txs, err := svc.ListTransactions(context.Background(), 100, 10, 0)
	require.NoError(t, err)
	assert.Len(t, txs, 2)
}

// TestWalletService_ListTransactions_ClampLimit 验证 limit 上限 / 下限。
func TestWalletService_ListTransactions_ClampLimit(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	_, err := svc.ListTransactions(context.Background(), 100, 0, 0)
	require.NoError(t, err)
	_, err = svc.ListTransactions(context.Background(), 100, 1000, 0)
	require.NoError(t, err)
}

// TestWalletService_InjectErrors 验证错误透传（防回归）。
func TestWalletService_InjectErrors(t *testing.T) {
	r := newFakeRepo()
	r.freezeErr = errors.New("pg down")
	svc := New(r)
	err := svc.OnOrderCompleted(context.Background(), 100, 1, decimal.NewFromInt(300))
	require.Error(t, err)
}

// TestWalletService_InjectErrors_Deduct 验证退款错误透传。
func TestWalletService_InjectErrors_Deduct(t *testing.T) {
	r := newFakeRepo()
	svc := New(r)
	require.NoError(t, svc.OnOrderCompleted(context.Background(), 100, 1, decimal.NewFromInt(300)))
	r.deductErr = errors.New("pg down")
	err := svc.OnRefund(context.Background(), 100, 1, decimal.NewFromInt(150))
	require.Error(t, err)
}