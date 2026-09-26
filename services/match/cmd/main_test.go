// Package main 内的 helper 函数（nilEscortLoader）单测。
package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNilEscortLoader_ReturnsNilNil 验证 nilEscortLoader 返回空 slice + nil error，
//
//	便于 match 在没接 escort 真实数据时也能跑路由（候选打分结果为空）。
func TestNilEscortLoader_ReturnsNilNil(t *testing.T) {
	var l nilEscortLoader
	got, err := l.ListAvailable(context.Background(), "beijing")
	assert.NoError(t, err)
	assert.Nil(t, got)
}