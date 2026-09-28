//go:build integration
// +build integration

// SwitchRole 集成测试：真 PG + 真 token 签发/校验。
//
// 用唯一 phone 前缀 +99000000xxx 隔离测试数据，每个 case 用 unique phone；
// 每个测试结束清理 test-created 行，不破坏 doctors DB 现有 users。
//
// 跑测：
//   go test -tags=integration -count=1 ./services/auth/internal/service/...
//
// 前置：docker compose up（PG 已就绪）+ migration 0016 users.roles JSONB 应用。
package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/auth/internal/repo"
	"github.com/growdu/doctors/shared/auth"
)

func testDSN2() string {
	if v := os.Getenv("DOCTORS_TEST_DSN"); v != "" {
		return v
	}
	return "postgres://doctors:doctors@127.0.0.1:5432/doctors?sslmode=disable"
}

// pgRepoAdapter 把 *repo.UserRepo（返回 *repo.User）适配成 service.UserRepo
// （返回 *service.User）。与 cmd/main.go 的 userRepoAdapter 等价，避免跨包依赖。
type pgRepoAdapter struct{ r *repo.UserRepo }

func newPgRepoAdapter(r *repo.UserRepo) *pgRepoAdapter { return &pgRepoAdapter{r: r} }

func (a *pgRepoAdapter) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	u := &repo.User{Phone: phone, Role: repo.Role(role), WxUnionID: unionid}
	if err := a.r.Create(ctx, u); err != nil {
		return 0, err
	}
	return u.ID, nil
}
func (a *pgRepoAdapter) FindByPhone(ctx context.Context, phone string) (*User, error) {
	u, err := a.r.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}
func (a *pgRepoAdapter) FindByUnionID(ctx context.Context, unionid string) (*User, error) {
	u, err := a.r.FindByUnionID(ctx, unionid)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}
func (a *pgRepoAdapter) FindByID(ctx context.Context, id int64) (*User, error) {
	u, err := a.r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toServiceUser(u), nil
}
func (a *pgRepoAdapter) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	return a.r.UpdateRealName(ctx, id, hash, tail)
}

func toServiceUser(u *repo.User) *User {
	return &User{
		ID:               u.ID,
		Phone:            u.Phone,
		Role:             string(u.Role),
		RealNameVerified: u.RealNameVerified,
	}
}

// setupRepoPool 起 PG 池 + 创建 fake SMS + Service。
//
//	每个 case 调用 t.Cleanup 删自己创建的 phone 行。
func setupRepoPool(t *testing.T, phone string) (*Service, *repo.UserRepo) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN2())
	require.NoError(t, err, "connect pg (docker compose up?)")

	repoR := repo.NewUserRepo(pool)
	adapter := newPgRepoAdapter(repoR)
	svc := &Service{
		repo:      adapter,
		sms:       &fakeSMS{codes: map[string]string{}},
		wx:        &fakeWX{},
		realName:  &fakeRealName{},
		jwtSecret: "integration-secret",
		jwtTTL:    time.Hour,
	}

	// 清理 hook：删测试 phone 关联的行
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanCancel()
		_, _ = pool.Exec(cleanCtx, "DELETE FROM users WHERE phone = $1", phone)
		pool.Close()
	})

	return svc, repoR
}

// newPhone 生成 139 开头 11 位唯一手机号（满足 sms.ValidatePhone 正则）。
//
//	取 UnixNano mod 1e7 保证 uniqueness。
func newPhone(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("139%08d", time.Now().UnixNano()%100000000)
}

// TestIntegration_SwitchRole_FullFlow 完整链路：注册→拿 token→DB 加 escort 角色
//
//	→ SwitchRole(token roles, active=escort) → 新 token active=escort。
func TestIntegration_SwitchRole_FullFlow(t *testing.T) {
	phone := newPhone(t)
	svc, repoR := setupRepoPool(t, phone)

	// 1. SMS 登录注册用户
	require.NoError(t, svc.SendSMS(context.Background(), phone))
	sender := svc.sms.(*fakeSMS)
	code := sender.codes[phone]
	require.NotEmpty(t, code)

	tok1, uid, err := svc.LoginBySMS(context.Background(), phone, code)
	require.NoError(t, err)
	require.NotZero(t, uid)

	// 2. DB 加 escort 角色（模拟运营审核通过 / 用户自助申请陪诊师）
	updated, err := repoR.AddRole(context.Background(), uid, repo.RoleEscort)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"patient", "escort"}, []string(updated), "roles 应含 patient+escort（顺序无关）")

	// 3. 用更新后的 roles 调 SwitchRole
	tok2, err := svc.SwitchRole(context.Background(), uid, []string(updated), "escort")
	require.NoError(t, err)
	require.NotEqual(t, tok1, tok2, "新 token 应不同于旧 token")

	// 4. 验证新 token claims
	claims, err := auth.Parse("integration-secret", tok2)
	require.NoError(t, err)
	assert.Equal(t, uid, claims.UserID)
	assert.ElementsMatch(t, []string{"patient", "escort"}, claims.Roles, "roles 含 patient+escort")
	assert.Equal(t, "escort", claims.Active)
	assert.True(t, claims.HasRole("patient"))
	assert.True(t, claims.HasRole("escort"))
	assert.False(t, claims.HasRole("admin"))
	assert.True(t, claims.ActiveIsValid())
}

// TestIntegration_SwitchRole_ActiveNotInDBRoles 验证 DB roles 不含 active → 拒绝。
func TestIntegration_SwitchRole_ActiveNotInDBRoles(t *testing.T) {
	phone := newPhone(t)
	svc, _ := setupRepoPool(t, phone)

	require.NoError(t, svc.SendSMS(context.Background(), phone))
	sender := svc.sms.(*fakeSMS)
	_, uid, err := svc.LoginBySMS(context.Background(), phone, sender.codes[phone])
	require.NoError(t, err)

	// 用户只有 patient role（注册默认）；切 escort 应失败
	_, err = svc.SwitchRole(context.Background(), uid, []string{"patient"}, "escort")
	assert.Error(t, err, "active=escort 不在 roles 应失败")
	assert.Contains(t, err.Error(), "active not in user's roles")
}

// TestIntegration_AddRole_RejectsDuplicates 验证 AddRole 重复添加不重复。
func TestIntegration_AddRole_RejectsDuplicates(t *testing.T) {
	phone := newPhone(t)
	svc, repoR := setupRepoPool(t, phone)

	require.NoError(t, svc.SendSMS(context.Background(), phone))
	sender := svc.sms.(*fakeSMS)
	_, uid, err := svc.LoginBySMS(context.Background(), phone, sender.codes[phone])
	require.NoError(t, err)

	r1, err := repoR.AddRole(context.Background(), uid, repo.RoleEscort)
	require.NoError(t, err)
	require.Len(t, r1, 2)

	r2, err := repoR.AddRole(context.Background(), uid, repo.RoleEscort)
	require.NoError(t, err)
	assert.Equal(t, r1, r2, "重复 AddRole 应返回相同 roles（distinct 后不变）")
	assert.Len(t, r2, 2, "去重后仍 2 个角色")
}

// TestIntegration_FindByPhone_ReadsRoles 验证 FindByID 读回 roles JSONB 正确。
func TestIntegration_FindByPhone_ReadsRoles(t *testing.T) {
	phone := newPhone(t)
	svc, repoR := setupRepoPool(t, phone)

	require.NoError(t, svc.SendSMS(context.Background(), phone))
	sender := svc.sms.(*fakeSMS)
	_, uid, err := svc.LoginBySMS(context.Background(), phone, sender.codes[phone])
	require.NoError(t, err)

	_, err = repoR.AddRole(context.Background(), uid, repo.RoleEscort)
	require.NoError(t, err)

	u, err := repoR.FindByID(context.Background(), uid)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, repo.RolePatient, u.Role, "v1 role 字段保留")
	assert.ElementsMatch(t, []string{"patient", "escort"}, []string(u.Roles))
}