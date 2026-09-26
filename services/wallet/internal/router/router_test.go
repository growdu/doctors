package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/wallet/internal/handler"
	"github.com/growdu/doctors/services/wallet/internal/repo"
)

// fakeSvc 实现 handler.Service 接口（最简）。
type fakeRouterSvc struct{}

func (fakeRouterSvc) GetWallet(ctx context.Context, userID int64) (*repo.Wallet, error) {
	return &repo.Wallet{UserID: userID, Currency: "CNY", Balance: decimal.Zero}, nil
}
func (fakeRouterSvc) CreateWithdrawal(ctx context.Context, userID int64, amount decimal.Decimal, channel, account string) (*repo.Withdrawal, error) {
	return &repo.Withdrawal{ID: 1, Amount: amount, Status: "pending", Channel: channel}, nil
}
func (fakeRouterSvc) ApproveWithdrawal(ctx context.Context, id, reviewerID int64) error { return nil }
func (fakeRouterSvc) MarkWithdrawalPaid(ctx context.Context, id int64) error            { return nil }
func (fakeRouterSvc) RejectWithdrawal(ctx context.Context, id, reviewerID int64, reason string) error {
	return nil
}
func (fakeRouterSvc) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*repo.Billing, error) {
	return nil, nil
}

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 用 fake 服务填充 handler；healthz 不依赖具体业务。
	h := handler.New(fakeRouterSvc{}, nil)
	r := New(h, "test-secret", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}