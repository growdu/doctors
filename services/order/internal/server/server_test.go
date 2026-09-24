package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/handler"
	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/services/order/internal/service"
)

type stubOrderRepo struct{}

func (stubOrderRepo) Create(ctx context.Context, o *repo.Order) error              { return nil }
func (stubOrderRepo) FindByID(ctx context.Context, id int64) (*repo.Order, error) { return nil, repo.ErrOrderNotFound }
func (stubOrderRepo) ListByPatient(ctx context.Context, id int64, l, o int) ([]*repo.Order, error) {
	return nil, nil
}
func (stubOrderRepo) UpdateStatus(ctx context.Context, id int64, to string, v int, e *int64) error {
	return nil
}
func (stubOrderRepo) InsertEvent(ctx context.Context, id int64, from *string, to string, a *int64, p []byte) error {
	return nil
}
func (stubOrderRepo) ListEvents(ctx context.Context, id int64) ([]*repo.OrderEvent, error) {
	return nil, nil
}
func (stubOrderRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
	return nil
}
func (stubOrderRepo) ConfirmByEscort(ctx context.Context, id int64, escortID int64, now time.Time, expectVersion int) error {
	return nil
}
func (stubOrderRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
	return nil
}
func (stubOrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
	return nil, nil
}

type stubUserLookup struct{}

func (stubUserLookup) FindByID(ctx context.Context, id int64) (*service.UserSnapshot, error) {
	return &service.UserSnapshot{ID: id, Role: "patient"}, nil
}

// TestServer_EngineExposesRouter 验证 Engine() 返回的 handler 可以处理 /healthz。
func TestServer_EngineExposesRouter(t *testing.T) {
	s := New(":0", handler.New(service.New(stubOrderRepo{}, stubUserLookup{})), "secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	s.Engine().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RunShutdownGracefully 启动后立刻取消 ctx，应在超时内返回 nil。
func TestServer_RunShutdownGracefully(t *testing.T) {
	s := New("127.0.0.1:0", handler.New(service.New(stubOrderRepo{}, stubUserLookup{})), "secret")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server.Run did not return within 3s after ctx cancel")
	}
}