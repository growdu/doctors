// Package main 内的 helper 函数（buildPool / nilRefundRepo）单测。
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/payment/internal/refund"
	"github.com/growdu/doctors/shared/config"
)

// TestBuildPool_EmptyDSNReturnsNil 验证 DSN 空时降级为 (nil, nil)。
func TestBuildPool_EmptyDSNReturnsNil(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = ""
	p, err := buildPool(cfg)
	assert.NoError(t, err)
	assert.Nil(t, p)
}

// TestBuildPool_InvalidDSNReturnsError 验证 DSN 不合法时返回 error。
func TestBuildPool_InvalidDSNReturnsError(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = "not-a-postgres-dsn"
	p, err := buildPool(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
}

// TestNilRefundRepo_ReturnsSentinel 验证 nilRefundRepo 所有方法返回相同的 sentinel。
func TestNilRefundRepo_ReturnsSentinel(t *testing.T) {
	var r nilRefundRepo
	ctx := context.Background()
	rec := &refund.Record{OrderID: 1, Amount: 100, Status: "created"}

	assert.True(t, errors.Is(r.Create(ctx, rec), errNilRefund))
	_, err := r.GetByOrderID(ctx, 1)
	assert.True(t, errors.Is(err, errNilRefund))
	assert.True(t, errors.Is(r.UpdateStatus(ctx, 1, "completed", "tx", ""), errNilRefund))
}

// TestNilRefundRepo_SentinelMessage 提醒运维配置 DSN。
func TestNilRefundRepo_SentinelMessage(t *testing.T) {
	assert.True(t, contains(errNilRefund.Error(), "dsn"))
}

// contains 是 strings.Contains 的本地别名（避免 import）。
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}