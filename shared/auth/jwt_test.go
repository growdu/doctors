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