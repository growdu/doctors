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
	"github.com/growdu/doctors/services/order/internal/service"
)

// TestServer_EngineExposesRouter 验证 Engine() 返回的 handler 可以处理 /healthz。
func TestServer_EngineExposesRouter(t *testing.T) {
	s := New(":0", handler.New(service.New()), "secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	s.Engine().ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RunShutdownGracefully 启动后立刻取消 ctx，应在超时内返回 nil。
func TestServer_RunShutdownGracefully(t *testing.T) {
	s := New("127.0.0.1:0", handler.New(service.New()), "secret")
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