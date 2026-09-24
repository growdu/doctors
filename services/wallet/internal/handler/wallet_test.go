package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/wallet/internal/repo"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// fakeSvc 实现 Service 接口。
type fakeSvc struct {
	wallet       *repo.Wallet
	walletErr    error
	withdrawal   *repo.Withdrawal
	withdrawErr  error
	txList       []*repo.Billing
	listErr      error
	approveErr   error
	payErr       error
	rejectErr    error
	approveCalls int
	payCalls     int
	rejectCalls  int
}

func (f *fakeSvc) GetWallet(ctx context.Context, userID int64) (*repo.Wallet, error) {
	return f.wallet, f.walletErr
}
func (f *fakeSvc) CreateWithdrawal(ctx context.Context, userID int64, amount decimal.Decimal, channel, account string) (*repo.Withdrawal, error) {
	return f.withdrawal, f.withdrawErr
}
func (f *fakeSvc) ApproveWithdrawal(ctx context.Context, id, reviewerID int64) error {
	f.approveCalls++
	return f.approveErr
}
func (f *fakeSvc) MarkWithdrawalPaid(ctx context.Context, id int64) error {
	f.payCalls++
	return f.payErr
}
func (f *fakeSvc) RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error {
	f.rejectCalls++
	return f.rejectErr
}
func (f *fakeSvc) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*repo.Billing, error) {
	return f.txList, f.listErr
}

// 强制 fakeSvc 满足 Service 接口（编译期）。
var _ Service = (*fakeSvc)(nil)
var _ AdminSvc = (*fakeSvc)(nil)

// fakeAuth 返回一个把 userID+role 塞进 ctx 的中间件（绕开 JWT 校验）。
func fakeAuth(uid int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Query("user_id") != "" {
			c.Set("user_id", uid)
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}

// TestGetUserWallet_OK 验证 GET /api/v1/users/me/wallet。
func TestGetUserWallet_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{wallet: &repo.Wallet{UserID: 1, Balance: decimal.NewFromInt(100), Frozen: decimal.NewFromInt(50), Currency: "CNY"}}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/wallet?user_id=1", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

// TestGetEscortWallet_OK 验证 GET /api/v1/escorts/me/wallet。
func TestGetEscortWallet_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{wallet: &repo.Wallet{UserID: 1, Balance: decimal.NewFromInt(200), Frozen: decimal.NewFromInt(800)}}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/escorts/me/wallet?user_id=1", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestCreateWithdrawal_OK 验证 POST /api/v1/escorts/me/wallet/withdraw。
func TestCreateWithdrawal_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{withdrawal: &repo.Withdrawal{ID: 1, Amount: decimal.NewFromInt(200), Status: "pending", Channel: "wx"}}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	body, _ := json.Marshal(map[string]any{"amount": "200.00", "channel": "wx", "account": "138****0000"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/escorts/me/wallet/withdraw?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestCreateWithdrawal_BelowMinimum 验证 < 100 元返回参数错误。
func TestCreateWithdrawal_BelowMinimum(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{withdrawErr: errs.New(errs.CodeParamInvalid, "最低提现金额 100 元")}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	body, _ := json.Marshal(map[string]any{"amount": "50.00", "channel": "wx", "account": "138****0000"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/escorts/me/wallet/withdraw?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	// httpx.Fail 总是 HTTP 200；业务码在 body.code 中。
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestCreateWithdrawal_BindError 验证 JSON 解析失败。
func TestCreateWithdrawal_BindError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/escorts/me/wallet/withdraw?user_id=1", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestCreateWithdrawal_AmountParseError 验证 amount 非数字。
func TestCreateWithdrawal_AmountParseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	body, _ := json.Marshal(map[string]any{"amount": "abc", "channel": "wx", "account": "138****0000"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/escorts/me/wallet/withdraw?user_id=1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// TestListTransactions_OK 验证 GET /api/v1/wallet/transactions。
func TestListTransactions_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{txList: []*repo.Billing{{ID: 1, Type: "order_income", Amount: decimal.NewFromInt(300)}}}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/wallet/transactions?user_id=1&limit=10&offset=0", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[[]map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data)
	assert.Len(t, resp.Data, 1)
}

// TestAdminApprove_OK 验证 admin approve endpoint。
func TestAdminApprove_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(999))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/wallet/withdrawals/123/approve?user_id=999", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, svc.approveCalls)
}

// TestAdminPay_OK 验证 admin pay endpoint。
func TestAdminPay_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(999))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/wallet/withdrawals/123/pay?user_id=999", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, svc.payCalls)
}

// TestAdminReject_OK 验证 admin reject endpoint。
func TestAdminReject_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(999))

	body, _ := json.Marshal(map[string]any{"reason": "bank card invalid"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/wallet/withdrawals/123/reject?user_id=999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, svc.rejectCalls)
}

// TestAll_NoToken_401 验证无 token 全部 401。
func TestAll_NoToken_401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeSvc{}
	h := New(svc, svc)
	r := gin.New()
	h.RegisterRoutes(r, fakeAuth(1))

	for _, url := range []string{
		"/api/v1/users/me/wallet",
		"/api/v1/escorts/me/wallet",
		"/api/v1/wallet/transactions",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		assert.Equal(t, http.StatusUnauthorized, w.Code, "no token → 401 at %s", url)
	}
}

// 引用 errors 包避免 unused 警告
var _ = errors.New