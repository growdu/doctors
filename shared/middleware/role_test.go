package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

const roleSecret = "role-secret"

func signTokenWithRole(t *testing.T, uid int64, role string) string {
	t.Helper()
	now := time.Now()
	tok, err := authpkg.Sign(roleSecret, authpkg.Claims{
		UserID: uid,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})
	require.NoError(t, err)
	return tok
}

// TestRoleAuth_NoToken 验证无 token → 401。
func TestRoleAuth_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", Auth(roleSecret, "uid", "role"), RoleAuth("super_admin"), func(c *gin.Context) {
		httpx.OK[any](c, nil)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p", nil))
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeUnauthorized), resp.Code)
}

// TestRoleAuth_RoleAllowed 验证 super_admin 命中白名单。
func TestRoleAuth_RoleAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", Auth(roleSecret, "uid", "role"), RoleAuth("super_admin", "order_admin"), func(c *gin.Context) {
		httpx.OK(c, gin.H{"role": c.GetString("role")})
	})
	tok := signTokenWithRole(t, 1, "super_admin")
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "super_admin", resp.Data["role"])
}

// TestRoleAuth_RoleForbidden 验证 viewer 调 force-cancel → 403 (11003)。
func TestRoleAuth_RoleForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", Auth(roleSecret, "uid", "role"), RoleAuth("super_admin", "order_admin"), func(c *gin.Context) {
		httpx.OK[any](c, nil)
	})
	tok := signTokenWithRole(t, 1, "viewer")
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeAdminForbidden), resp.Code)
}

// TestRoleAuth_MultipleRoles 验证多角色白名单 OR 命中。
func TestRoleAuth_MultipleRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", Auth(roleSecret, "uid", "role"), RoleAuth("refund_admin", "super_admin"), func(c *gin.Context) {
		httpx.OK[any](c, nil)
	})
	tok := signTokenWithRole(t, 2, "refund_admin")
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}