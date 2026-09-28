package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/auth"
)

func TestSignAndParse_RoundTrip(t *testing.T) {
	// Arrange
	const secret = "test-secret"
	claims := auth.Claims{
		UserID:  42,
		Role:    "patient",
		UnionID: "union-xyz",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Act
	tok, err := auth.Sign(secret, claims)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	got, err := auth.Parse(secret, tok)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(42), got.UserID)
	assert.Equal(t, "patient", got.Role)
	assert.Equal(t, "union-xyz", got.UnionID)
}

func TestParse_WrongSecretFails(t *testing.T) {
	// Arrange
	tok, err := auth.Sign("secret-A", auth.Claims{
		UserID: 1, Role: "patient",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	require.NoError(t, err)

	// Act
	_, err = auth.Parse("secret-B", tok)

	// Assert
	assert.Error(t, err, "用错误密钥签名/解析应失败")
}

func TestParse_ExpiredTokenFails(t *testing.T) {
	// Arrange
	tok, err := auth.Sign("s", auth.Claims{
		UserID: 1, Role: "patient",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	require.NoError(t, err)

	// Act
	_, err = auth.Parse("s", tok)

	// Assert
	assert.Error(t, err)
}

func TestSign_EmptySecretRejected(t *testing.T) {
	_, err := auth.Sign("", auth.Claims{UserID: 1, Role: "patient"})
	assert.Error(t, err, "空密钥必须拒绝")
}

func TestParse_MalformedTokenFails(t *testing.T) {
	_, err := auth.Parse("s", "not.a.token")
	assert.Error(t, err)
}

func TestSignAndParse_IncludesIssuer(t *testing.T) {
	tok, err := auth.Sign("s", auth.Claims{
		UserID: 1, Role: "escort",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "doctors",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	require.NoError(t, err)

	got, err := auth.Parse("s", tok)
	require.NoError(t, err)
	assert.Equal(t, "doctors", got.Issuer)
}

// TestSignAndParse_RolesAndActive_RoundTrip v2 多角色并发 + 当前激活角色
//
//	Patient 兼职陪诊师场景：roles=["patient","escort"]，active="escort"
//	切换 active 不改 roles，前后端按需展示对应域 UI。
func TestSignAndParse_RolesAndActive_RoundTrip(t *testing.T) {
	const secret = "test-secret"
	claims := auth.Claims{
		UserID: 100,
		Roles:   []string{"patient", "escort"},
		Active:  "escort",
		Role:    "escort", // v1 兼容字段：取 active
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tok, err := auth.Sign(secret, claims)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	got, err := auth.Parse(secret, tok)
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.UserID)
	assert.Equal(t, []string{"patient", "escort"}, got.Roles)
	assert.Equal(t, "escort", got.Active)
	assert.True(t, got.HasRole("patient"), "兼职用户应同时含 patient 角色")
	assert.True(t, got.HasRole("escort"))
	assert.False(t, got.HasRole("admin"), "不存在的角色应 false")
}

// TestHasRole_EmptyRolesReturnsFalse 验证零值（roles 数组未设）时所有 HasRole 返回 false。
func TestHasRole_EmptyRolesReturnsFalse(t *testing.T) {
	c := &auth.Claims{UserID: 1}
	assert.False(t, c.HasRole("patient"))
	assert.False(t, c.HasRole("escort"))
	assert.False(t, c.HasRole("admin"))
}

// TestActiveIsValid 验证 Active 字段合法性检查（必须在 Roles 中，否则视为非法激活）。
func TestActiveIsValid(t *testing.T) {
	// 正常：active ∈ roles
	c := &auth.Claims{Roles: []string{"patient", "escort"}, Active: "escort"}
	assert.True(t, c.ActiveIsValid())

	// 异常：active 不在 roles
	c2 := &auth.Claims{Roles: []string{"patient"}, Active: "admin"}
	assert.False(t, c2.ActiveIsValid(), "active 不在 roles 应判为非法")

	// 兼容：v1 仅 Role 字段（无 Roles），Active 应回退到 Role
	c3 := &auth.Claims{Role: "patient"}
	assert.True(t, c3.ActiveIsValid(), "v1 单 Role 应兼容 ActiveIsValid")

	// 异常：v1 也空
	c4 := &auth.Claims{}
	assert.False(t, c4.ActiveIsValid())
}