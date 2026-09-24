package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/httpx"
)

const testSecret = "test-secret"

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Auth(testSecret), func(c *gin.Context) {
		httpx.OK(c, gin.H{"uid": UserID(c), "role": Role(c)})
	})
	return r
}

// TestAuth_MissingHeader 未带 Authorization 头应被拦截。
func TestAuth_MissingHeader(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/protected", nil))

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

// TestAuth_WrongScheme 非 Bearer 前缀被拒。
func TestAuth_WrongScheme(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}

// TestAuth_ValidToken 合法 token 写入 ctx。
func TestAuth_ValidToken(t *testing.T) {
	tok, err := authpkg.Sign(testSecret, authpkg.Claims{UserID: 42, Role: "patient"})
	require.NoError(t, err)

	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.EqualValues(t, 42, resp.Data["uid"])
	assert.Equal(t, "patient", resp.Data["role"])
}

// TestAuth_BadSignature 错误签名 token 被拒。
func TestAuth_BadSignature(t *testing.T) {
	tok, err := authpkg.Sign("wrong-secret", authpkg.Claims{UserID: 1, Role: "patient"})
	require.NoError(t, err)

	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEqual(t, 0, resp.Code)
}