// Package repo 提供 auth-service 的数据访问层。
//
// 设计要点：
//   - 用 pgx 直接写 SQL，避免引入 sqlc 工具链；与 shared/db 风格一致。
//   - 不带业务校验；只做参数绑定、错误翻译、字段映射。
//   - 用户不存在统一返回 ErrUserNotFound，便于上层 errors.Is。
//   - 每条 UPDATE 自动维护 updated_at。
package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Role 是用户的角色枚举，与 users.role CHECK 约束对齐。
type Role string

const (
	RolePatient Role = "patient"
	RoleEscort  Role = "escort"
	RoleAdmin   Role = "admin"
)

// User 映射 users 表行。
type User struct {
	ID               int64
	Phone            string
	Role             Role
	Nickname         *string
	AvatarURL        *string
	RealNameVerified bool
	IDCardHash       *string
	IDCardTail       *string
	WxUnionID        *string
	WxOpenIDMini     *string
	WxOpenIDApp      *string
	Status           int16
}

// ErrUserNotFound 是查询无结果时的哨兵错误。
var ErrUserNotFound = errors.New("repo: user not found")

// UserRepo 是 users 表的仓储对象。
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo 构造仓储；pool 由调用方负责生命周期。
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create 插入新用户；ID / Status 由 DB 默认填入并回写到 u。
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	const q = `
		INSERT INTO users (phone, role, nickname, avatar_url, wx_unionid, wx_openid_mini, wx_openid_app)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status`
	return r.pool.QueryRow(ctx, q,
		u.Phone, u.Role, u.Nickname, u.AvatarURL, u.WxUnionID, u.WxOpenIDMini, u.WxOpenIDApp,
	).Scan(&u.ID, &u.Status)
}

// FindByPhone 按手机号查找；未命中 ErrUserNotFound。
func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*User, error) {
	const q = baseSelect + ` WHERE phone = $1 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, phone))
}

// FindByUnionID 按 unionid 查找；未命中 ErrUserNotFound。
func (r *UserRepo) FindByUnionID(ctx context.Context, unionid string) (*User, error) {
	const q = baseSelect + ` WHERE wx_unionid = $1 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, unionid))
}

// FindByID 按主键查找；未命中 ErrUserNotFound。
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	const q = baseSelect + ` WHERE id = $1 AND deleted_at IS NULL`
	return r.scanOne(r.pool.QueryRow(ctx, q, id))
}

// UpdateRealName 设置实名哈希与末四位，并标记 real_name_verified=true。
func (r *UserRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	const q = `
		UPDATE users SET id_card_hash = $1, id_card_tail = $2,
		                 real_name_verified = TRUE, updated_at = NOW()
		 WHERE id = $3 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, hash, tail, id)
	if err != nil {
		return fmt.Errorf("user repo: update real name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// baseSelect 是 SELECT 子句，WHERE 由调用方追加。
const baseSelect = `
	SELECT id, phone, role, nickname, avatar_url, real_name_verified,
	       id_card_hash, id_card_tail, wx_unionid, wx_openid_mini, wx_openid_app, status
	FROM users`

// scanOne 把 pgx.Row 扫描为 *User；pgx.ErrNoRows 翻译为 ErrUserNotFound。
func (r *UserRepo) scanOne(row pgx.Row) (*User, error) {
	var (
		u           User
		nickname    *string
		avatarURL   *string
		idHash      *string
		idTail      *string
		unionid     *string
		openidMini  *string
		openidApp   *string
	)
	err := row.Scan(
		&u.ID, &u.Phone, &u.Role, &nickname, &avatarURL, &u.RealNameVerified,
		&idHash, &idTail, &unionid, &openidMini, &openidApp, &u.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repo: scan: %w", err)
	}
	u.Nickname = nickname
	u.AvatarURL = avatarURL
	u.IDCardHash = idHash
	u.IDCardTail = idTail
	u.WxUnionID = unionid
	u.WxOpenIDMini = openidMini
	u.WxOpenIDApp = openidApp
	return &u, nil
}