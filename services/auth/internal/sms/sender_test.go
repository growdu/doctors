package sms

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogSender_RecordsCode 验证 default mock 把验证码记入 in-memory map。
func TestLogSender_RecordsCode(t *testing.T) {
	s := NewLogSender()
	err := s.Send(context.Background(), "13800138000", "654321")
	require.NoError(t, err)

	got, ok := s.LookupCode("13800138000")
	require.True(t, ok)
	assert.Equal(t, "654321", got)
}

// TestLogSender_OverwritesCode 同一手机号再次发送会覆盖旧值。
func TestLogSender_OverwritesCode(t *testing.T) {
	s := NewLogSender()
	require.NoError(t, s.Send(context.Background(), "13800138000", "111111"))
	require.NoError(t, s.Send(context.Background(), "13800138000", "222222"))
	got, _ := s.LookupCode("13800138000")
	assert.Equal(t, "222222", got)
}

// TestLogSender_LookupMissing 手机号不存在返回 ok=false。
func TestLogSender_LookupMissing(t *testing.T) {
	s := NewLogSender()
	_, ok := s.LookupCode("111")
	assert.False(t, ok)
}

// TestLogSender_EmitsLog 验证日志输出含 phone 与 code（开发期便于查）。
func TestLogSender_EmitsLog(t *testing.T) {
	s := NewLogSender()
	// capture stderr via package-level writer
	old := sw
	defer func() { sw = old }()

	buf := newBuffer()
	sw = buf
	require.NoError(t, s.Send(context.Background(), "13800138000", "123456"))

	out := buf.String()
	assert.True(t, strings.Contains(out, "13800138000"))
	assert.True(t, strings.Contains(out, "123456"))
}

// TestValidatePhone 表驱动测试手机号格式校验。
func TestValidatePhone(t *testing.T) {
	cases := []struct {
		phone string
		ok    bool
	}{
		{"13800138000", true},
		{"+8613800138000", true},
		{"1380013800", false},   // 太短
		{"138001380000", false}, // 太长
		{"abcdefghijk", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.phone, func(t *testing.T) {
			assert.Equal(t, c.ok, ValidatePhone(c.phone))
		})
	}
}