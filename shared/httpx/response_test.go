package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/httpx"
)

// newCtx 构造一个带 GET /test 请求的测试 Context。
func newCtx(t *testing.T, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

func TestOK_ReturnsJSONWithCodeZero(t *testing.T) {
	// Arrange
	c, w := newCtx(t, nil)
	type payload struct {
		Name string `json:"name"`
	}
	want := payload{Name: "alice"}

	// Act
	httpx.OK(c, want)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var got httpx.Resp[payload]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 0, got.Code)
	assert.Equal(t, "ok", got.Message)
	assert.Equal(t, "alice", got.Data.Name)
	assert.NotEmpty(t, got.TraceID)
}

func TestOK_HandlesNilData(t *testing.T) {
	// Arrange
	c, w := newCtx(t, nil)

	// Act
	httpx.OK[any](c, nil)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)

	var got httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 0, got.Code)
	assert.Nil(t, got.Data)
}

func TestFail_SetsNonZeroCode(t *testing.T) {
	// Arrange
	c, w := newCtx(t, nil)

	// Act
	httpx.Fail(c, 10001, "参数无效")

	// Assert
	require.Equal(t, http.StatusOK, w.Code, "Fail 始终返回 200，业务码在 body 里")

	var got httpx.Resp[any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 10001, got.Code)
	assert.Equal(t, "参数无效", got.Message)
	assert.Nil(t, got.Data)
}

func TestOK_ReusesExistingTraceID(t *testing.T) {
	// Arrange
	c, w := newCtx(t, nil)
	const traceID = "trace-abc-123"
	c.Set(httpx.TraceIDKey, traceID)

	// Act
	httpx.OK(c, map[string]int{"x": 1})

	// Assert
	var got httpx.Resp[map[string]int]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, traceID, got.TraceID, "应当复用已有 trace_id")
}

func TestOK_UsesHeaderTraceIDWhenSet(t *testing.T) {
	// Arrange
	c, w := newCtx(t, map[string]string{httpx.HeaderTraceID: "trace-from-header"})
	// Act
	httpx.OK(c, 1)

	// Assert
	var got httpx.Resp[int]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "trace-from-header", got.TraceID)
}