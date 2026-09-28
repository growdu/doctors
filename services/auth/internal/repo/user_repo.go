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
	"database/sql/driver"
	"encoding/json"
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

// RolesSlice 是用户多角色数组（v2 unified-app），实现 sql.Scanner / driver.Valuer
//
//	适配 pgx 读写 JSONB 列（users.roles）。
type RolesSlice []string

// Value 实现 driver.Valuer；空切片存为 "[]"（与列 default 一致）。
func (r RolesSlice) Value() (driver.Value, error) {
	if r == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(r))
}

// Scan 实现 sql.Scanner；从 JSONB 解析到 []string。
func (r *RolesSlice) Scan(src any) error {
	if src == nil {
		*r = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("repo: RolesSlice.Scan unsupported src type %T", src)
	}
	if len(b) == 0 {
		*r = nil
		return nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("repo: RolesSlice.Scan unmarshal: %w", err)
	}
	*r = out
	return nil
}

// User 映射 users 表行。
type User struct {
	ID               int64
	Phone            string
	Role             Role
	Roles            RolesSlice // v2：多角色并发（兼职 patient+escort）
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
// v2：roles 字段如果非空写入 JSONB；列 DEFAULT '[]' 保证 nil 也安全。
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	roles := u.Roles
	if roles == nil {
		roles = RolesSlice{string(u.Role)}
	}
	const q = `
		INSERT INTO users (phone, role, roles, nickname, avatar_url, wx_unionid, wx_openid_mini, wx_openid_app)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status`
	return r.pool.QueryRow(ctx, q,
		u.Phone, u.Role, roles, u.Nickname, u.AvatarURL, u.WxUnionID, u.WxOpenIDMini, u.WxOpenIDApp,
	).Scan(&u.ID, &u.Status)
}

// AddRole 给用户追加一个角色（v2 multi-role）；不重复添加。
//
//	若新角色已存在则 no-op；成功写入后从 DB 拉回最新 roles 给调用方同步本地缓存。
func (r *UserRepo) AddRole(ctx context.Context, userID int64, role Role) (RolesSlice, error) {
	const q = `
		UPDATE users
		SET roles = (
			SELECT jsonb_agg(DISTINCT v)
			FROM (
				SELECT jsonb_array_elements_text(
					CASE WHEN jsonb_typeof(roles) = 'array' THEN roles ELSE '[]'::jsonb END
				) AS v
				UNION ALL
				SELECT $2::text AS v
			) AS combined
		),
		updated_at = NOW()
		WHERE id = $1
		RETURNING roles`
	var out RolesSlice
	err := r.pool.QueryRow(ctx, q, userID, string(role)).Scan(&out)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user repo: add role: %w", err)
	}
	return out, nil
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
	SELECT id, phone, role, roles, nickname, avatar_url, real_name_verified,
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
		&u.ID, &u.Phone, &u.Role, &u.Roles, &nickname, &avatarURL, &u.RealNameVerified,
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