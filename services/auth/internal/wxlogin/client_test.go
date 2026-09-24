package wxlogin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClient_Code2Session 表驱动测试 code → unionid/openid 的映射。
// mock 把 "wx-mock-" 前缀剥掉，后缀直接用作 unionid / openid。
func TestMockClient_Code2Session(t *testing.T) {
	c := NewMockClient()
	cases := []struct {
		code    string
		unionid string
		openid  string
	}{
		{"wx-mock-A", "unionid-A", "openid-A"},
		{"wx-mock-B", "unionid-B", "openid-B"},
		{"wx-mock-anything", "unionid-anything", "openid-anything"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			u, o, err := c.Code2Session(context.Background(), tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.unionid, u)
			assert.Equal(t, tc.openid, o)
		})
	}
}

// TestMockClient_EmptyCode 校验空 code 返回错误。
func TestMockClient_EmptyCode(t *testing.T) {
	c := NewMockClient()
	_, _, err := c.Code2Session(context.Background(), "")
	assert.Error(t, err)
}

// TestMockClient_StableOutput 同一 Code 多次调用结果一致。
func TestMockClient_StableOutput(t *testing.T) {
	c := NewMockClient()
	u1, o1, _ := c.Code2Session(context.Background(), "code-x")
	u2, o2, _ := c.Code2Session(context.Background(), "code-x")
	assert.Equal(t, u1, u2)
	assert.Equal(t, o1, o2)
}