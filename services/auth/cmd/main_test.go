// Package main 内的 helper 函数（buildPool / adapter / nilUserRepo）单测。
//
// 注：main() 本身难单测（涉及 tracer / signal / http server），下面的用例只覆盖
//
//	辅助逻辑，确保 §31 PG 接入后退化路径 / 适配层行为稳定。
package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/services/auth/internal/repo"
	"github.com/growdu/doctors/services/auth/internal/service"
	"github.com/growdu/doctors/shared/config"
)

// TestBuildPool_EmptyDSNReturnsNil 验证 DSN 空时降级为 (nil, nil)，
//
//	不抛 error，方便 main 在没配置 DB 的开发场景下继续启动。
func TestBuildPool_EmptyDSNReturnsNil(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = ""
	p, err := buildPool(cfg)
	assert.NoError(t, err)
	assert.Nil(t, p)
}

// TestBuildPool_InvalidDSNReturnsError 验证 DSN 不合法时返回 error，
//
//	阻止带错误配置的进程继续启动。
func TestBuildPool_InvalidDSNReturnsError(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.DSN = "not-a-postgres-dsn"
	p, err := buildPool(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
}

// TestNilUserRepo_ReturnsSentinel 验证 nilUserRepo 所有方法都返回
//
//	相同的 sentinel error，便于上层 errors.Is 判断"未接通 PG"。
func TestNilUserRepo_ReturnsSentinel(t *testing.T) {
	var r nilUserRepo
	ctx := context.Background()

	_, err := r.Create(ctx, "13800138000", "patient", nil)
	assert.True(t, errors.Is(err, errNilUserRepo), "Create 应返回 errNilUserRepo")

	_, err = r.FindByPhone(ctx, "x")
	assert.True(t, errors.Is(err, errNilUserRepo))

	_, err = r.FindByUnionID(ctx, "x")
	assert.True(t, errors.Is(err, errNilUserRepo))

	_, err = r.FindByID(ctx, 1)
	assert.True(t, errors.Is(err, errNilUserRepo))

	err = r.UpdateRealName(ctx, 1, "h", "t")
	assert.True(t, errors.Is(err, errNilUserRepo))
}

// TestNilUserRepo_SentinelMessage 检查 sentinel 文案，提醒运维配置 DSN。
func TestNilUserRepo_SentinelMessage(t *testing.T) {
	assert.True(t, strings.Contains(errNilUserRepo.Error(), "dsn"),
		"sentinel error 应提示 DSN 配置缺失，便于定位")
}

// TestUserRepoAdapter_NilPoolOK 验证 adapter 构造不依赖 pool 是否为 nil
//
//	（实际使用时 pool 必然非 nil；这里验证 adapter 自身可独立构造）。
func TestUserRepoAdapter_NilPoolOK(t *testing.T) {
	// pool=nil 也能 NewUserRepo；adapter 只是把 *UserRepo 包一层。
	r := repo.NewUserRepo((*pgxpool.Pool)(nil))
	a := newUserRepoAdapter(r)
	assert.NotNil(t, a)
	// 真正的 DB 调用会 panic（pool nil），这里仅校验类型契约实现。
	var _ service.UserRepo = a
}