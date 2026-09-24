package realname

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockVerifier_ValidReturnsHash 验证合法身份证返回 hash + 末四位。
func TestMockVerifier_ValidReturnsHash(t *testing.T) {
	v := NewMockVerifier("test-salt")
	ok, hash, tail, err := v.Verify(context.Background(), "张三", "110101199001011234")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, hash, 64, "sha256 hex 应为 64")
	assert.Equal(t, "1234", tail)
}

// TestMockVerifier_EmptyFields 校验空字段被拒。
func TestMockVerifier_EmptyFields(t *testing.T) {
	v := NewMockVerifier("salt")
	cases := []struct{ name, id string }{
		{"", "110101199001011234"},
		{"张三", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/"+tc.id, func(t *testing.T) {
			ok, _, _, err := v.Verify(context.Background(), tc.name, tc.id)
			assert.False(t, ok)
			assert.Error(t, err)
		})
	}
}

// TestMockVerifier_TailTooShort 校验身份证少于 4 位被拒。
func TestMockVerifier_TailTooShort(t *testing.T) {
	v := NewMockVerifier("salt")
	ok, _, _, err := v.Verify(context.Background(), "张三", "123")
	assert.False(t, ok)
	assert.Error(t, err)
}

// TestMockVerifier_DeterministicHash 同一身份证 + salt 多次调用结果一致。
func TestMockVerifier_DeterministicHash(t *testing.T) {
	v := NewMockVerifier("salt-x")
	_, h1, _, _ := v.Verify(context.Background(), "李四", "110101199001011235")
	_, h2, _, _ := v.Verify(context.Background(), "李四", "110101199001011235")
	assert.Equal(t, h1, h2)
}

// TestMockVerifier_DifferentSaltDifferentHash 不同 salt → 不同 hash。
func TestMockVerifier_DifferentSaltDifferentHash(t *testing.T) {
	v1 := NewMockVerifier("salt-1")
	v2 := NewMockVerifier("salt-2")
	_, h1, _, _ := v1.Verify(context.Background(), "李四", "110101199001011235")
	_, h2, _, _ := v2.Verify(context.Background(), "李四", "110101199001011235")
	assert.NotEqual(t, h1, h2)
}

// TestMockVerifier_DoesNotLeakIDCard 验证返回里不含明文身份证号。
func TestMockVerifier_DoesNotLeakIDCard(t *testing.T) {
	v := NewMockVerifier("salt")
	id := "110101199001011234"
	_, hash, _, _ := v.Verify(context.Background(), "张三", id)
	assert.False(t, strings.Contains(hash, id))
}