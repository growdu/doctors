package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/auth/internal/handler"
	"github.com/growdu/doctors/services/auth/internal/service"
)

// stub deps 满足 service 各接口；最小可用。
type stubRepo struct{}

func (r *stubRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	return 1, nil
}
func (r *stubRepo) FindByPhone(ctx context.Context, phone string) (*service.User, error) {
	return &service.User{ID: 1, Phone: phone, Role: "patient"}, nil
}
func (r *stubRepo) FindByUnionID(ctx context.Context, unionid string) (*service.User, error) {
	return &service.User{ID: 1, Role: "patient"}, nil
}
func (r *stubRepo) FindByID(ctx context.Context, id int64) (*service.User, error) {
	return &service.User{ID: id, Phone: "13800138000", Role: "patient"}, nil
}
func (r *stubRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return nil
}
type stubSMS struct{}

func (s *stubSMS) Send(ctx context.Context, phone, code string) error { return nil }
func (s *stubSMS) VerifyCode(phone, code string) bool                 { return true }
type stubWX struct{}

func (w *stubWX) Code2Session(ctx context.Context, code string) (string, string, error) {
	return "u-" + code, "o-" + code, nil
}
type stubRN struct{}

func (r *stubRN) Verify(ctx context.Context, name, idCard string) (bool, string, string, error) {
	return true, "h", "1234", nil
}

// TestServer_EngineExposesRouter 验证 Engine() 返回的 handler 可以处理 /healthz。
func TestServer_EngineExposesRouter(t *testing.T) {
	svc := service.New(&stubRepo{}, &stubSMS{}, &stubWX{}, &stubRN{}, "secret", time.Minute)
	h := handler.New(svc)
	s := New(":0", h, "secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	s.Engine().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RunShutdownGracefully 启动后立刻取消 ctx，应在超时内返回 nil。
func TestServer_RunShutdownGracefully(t *testing.T) {
	svc := service.New(&stubRepo{}, &stubSMS{}, &stubWX{}, &stubRN{}, "secret", time.Minute)
	h := handler.New(svc)
	s := New("127.0.0.1:0", h, "secret")

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