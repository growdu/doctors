// Package main 内的 helper 函数（buildPool / nil*Repo）单测。
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

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

// TestNilProfileRepo_ReturnsSentinel 验证 nilProfileRepo 所有方法返回 errNil。
func TestNilProfileRepo_ReturnsSentinel(t *testing.T) {
	var r nilProfileRepo
	ctx := context.Background()
	_, err := r.GetByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(r.UpdateNickname(ctx, 1, "n"), errNil))
	assert.True(t, errors.Is(r.UpdateAvatar(ctx, 1, "u"), errNil))
}

// TestNilRepos_AllSentinel 验证所有 nil 占位 repo 都返回 errNil。
func TestNilRepos_AllSentinel(t *testing.T) {
	ctx := context.Background()

	// address
	var ar nilAddrRepo
	assert.True(t, errors.Is(ar.Create(ctx, nil), errNil))
	assert.True(t, errors.Is(ar.CreateDefault(ctx, nil), errNil))
	_, err := ar.ListByUser(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	_, err = ar.CountByUser(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(ar.Delete(ctx, 1, 1), errNil))

	// coupon
	var cr nilCouponRepo
	_, err = cr.ListActive(ctx, 10, 0)
	assert.True(t, errors.Is(err, errNil))
	assert.True(t, errors.Is(cr.MarkUsed(ctx, 1, 1), errNil))

	// hospital（只测 GetByID；List 需 hospital.ListFilter{}）
	var hr nilHospitalRepo
	_, err = hr.GetByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))

	// pkg
	var pr nilPackageRepo
	_, err = pr.ListByHospital(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
	_, err = pr.GetByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))

	// virtualnumber（只测 GetByID；Allocate / Release 需具体入参）
	var vr nilVNRepo
	_, err = vr.GetByID(ctx, 1)
	assert.True(t, errors.Is(err, errNil))
}