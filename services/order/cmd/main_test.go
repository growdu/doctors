// Package main 内的 helper 函数（buildPool / nilOrderRepo）单测。
package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/order/internal/repo"
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

// TestNilOrderRepo_ReturnsSentinel 验证 nilOrderRepo 所有方法都返回相同的 sentinel error。
func TestNilOrderRepo_ReturnsSentinel(t *testing.T) {
	var r nilOrderRepo
	ctx := context.Background()
	o := &repo.Order{OrderNo: "x", PatientID: 1, HospitalID: 1, PackageID: 1, ServiceStartAt: time.Now(), Amount: 1, FinalAmount: 1, Status: "pending_payment"}
	from := "x"
	actor := int64(1)

	assert.True(t, errors.Is(r.Create(ctx, o), errNilOrderRepo))
	_, err := r.FindByID(ctx, 1)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	_, err = r.ListByPatient(ctx, 1, 10, 0)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	_, err = r.ListByEscort(ctx, 1, "", 10, 0)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	assert.True(t, errors.Is(r.UpdateStatus(ctx, 1, "x", 1, nil), errNilOrderRepo))
	assert.True(t, errors.Is(r.InsertEvent(ctx, 1, &from, "x", &actor, nil), errNilOrderRepo))
	_, err = r.ListEvents(ctx, 1)
	assert.True(t, errors.Is(err, errNilOrderRepo))
	assert.True(t, errors.Is(r.SelectForEscort(ctx, 1, 1, time.Now(), 1), errNilOrderRepo))
	assert.True(t, errors.Is(r.ConfirmByEscort(ctx, 1, 1, time.Now(), 1), errNilOrderRepo))
	assert.True(t, errors.Is(r.RejectByEscort(ctx, 1, 1, 1), errNilOrderRepo))
	_, err = r.PendingExpired(ctx, time.Now(), 10)
	assert.True(t, errors.Is(err, errNilOrderRepo))
}

// TestNilOrderRepo_SentinelMessage 检查 sentinel 文案，提醒运维配置 DSN。
func TestNilOrderRepo_SentinelMessage(t *testing.T) {
	assert.True(t, strings.Contains(errNilOrderRepo.Error(), "dsn"),
		"sentinel error 应提示 DSN 配置缺失")
}