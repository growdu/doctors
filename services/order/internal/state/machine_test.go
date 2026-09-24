package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCanTransition_Legal 表驱动覆盖所有合法转换。
func TestCanTransition_Legal(t *testing.T) {
	cases := []struct{ from, to Status }{
		{StatusCreated, StatusPaid},
		{StatusCreated, StatusCanceled},
		{StatusPaid, StatusMatching},
		{StatusPaid, StatusCanceled},
		{StatusMatching, StatusAccepted},
		{StatusMatching, StatusCanceled},
		{StatusAccepted, StatusInService},
		{StatusAccepted, StatusCanceled},
		{StatusAccepted, StatusMatching},
		{StatusInService, StatusCompleted},
		{StatusCompleted, StatusReviewed},
		{StatusCompleted, StatusRefunding},
		{StatusReviewed, StatusClosed},
		{StatusRefunding, StatusRefunded},
		{StatusRefunded, StatusClosed},
	}
	for _, c := range cases {
		t.Run(string(c.from)+"->"+string(c.to), func(t *testing.T) {
			assert.True(t, CanTransition(c.from, c.to))
		})
	}
}

// TestCanTransition_Illegal 表驱动覆盖非法转换。
func TestCanTransition_Illegal(t *testing.T) {
	cases := []struct{ from, to Status }{
		{StatusCreated, StatusAccepted},     // 跳过 paid/matching
		{StatusPaid, StatusAccepted},        // 跳过 matching
		{StatusCompleted, StatusAccepted},   // 反向
		{StatusClosed, StatusCreated},       // 终态不能再走
		{StatusCanceled, StatusPaid},        // 取消后不能再支付
		{StatusRefunded, StatusRefunding},   // 终态
		{Status("" + "garbage"), StatusPaid}, // 非法 from
	}
	for _, c := range cases {
		t.Run(string(c.from)+"->"+string(c.to), func(t *testing.T) {
			assert.False(t, CanTransition(c.from, c.to))
		})
	}
}

// TestIsTerminal 终态：closed / canceled 不再有出向边。
func TestIsTerminal(t *testing.T) {
	assert.True(t, IsTerminal(StatusClosed))
	assert.True(t, IsTerminal(StatusCanceled))
	assert.False(t, IsTerminal(StatusPaid))
	assert.False(t, IsTerminal(StatusAccepted))
}

// TestIsValid 校验字符串是否在状态枚举内。
func TestIsValid(t *testing.T) {
	assert.True(t, IsValid(StatusCreated))
	assert.True(t, IsValid(StatusRefunded))
	assert.False(t, IsValid(Status("nope")))
	assert.False(t, IsValid(Status("")))
}