// Package hospital 实现 user-service 医院库的数据访问层。
//
// 设计要点：
//   - v1 只读 + 简单查询：List（按 city_id + status + keyword）+ GetByID。
//   - 无越权问题（公共资源，不带 user_id）。
//   - city_id 为 0 视作"全部城市"。
package hospital

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record 映射 hospitals 表行。
type Record struct {
	ID          int64
	Name        string
	CityID      int64
	Level       string  // 3a | 3b | 2a | 2b | 1 | other
	Status      string  // active | inactive
	Address     string
	Lat         *float64
	Lng         *float64
	Phone       *string
	Departments *string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ErrNotFound 是查询无结果哨兵。
var ErrNotFound = errors.New("hospital: not found")

// Repo 是 hospitals 表的仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const baseSelect = `
	SELECT id, name, city_id, level, status, address,
	       lat, lng, phone, departments, description,
	       created_at, updated_at
	FROM hospitals`

// ListFilter 是列表过滤参数。
type ListFilter struct {
	CityID  int64  // 0 = 全部
	Level   string // 空 = 全部
	Keyword string // 名称 ILIKE %keyword%；空 = 全部
	Status  string // 空 = 仅 active
	Limit   int
	Offset  int
}

// List 按 filter 查医院列表 + 总数。
func (r *Repo) List(ctx context.Context, f ListFilter) ([]*Record, int, error) {
	if f.Status == "" {
		f.Status = "active"
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	// 构造 WHERE
	var where []string
	var args []any
	idx := 1
	if f.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.CityID > 0 {
		where = append(where, fmt.Sprintf("city_id = $%d", idx))
		args = append(args, f.CityID)
		idx++
	}
	if f.Level != "" {
		where = append(where, fmt.Sprintf("level = $%d", idx))
		args = append(args, f.Level)
		idx++
	}
	if k := strings.TrimSpace(f.Keyword); k != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", idx))
		args = append(args, "%"+k+"%")
		idx++
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	// total
	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM hospitals "+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count hospitals: %w", err)
	}

	// list
	listSQL := baseSelect + " " + whereSQL +
		fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", idx, idx+1)
	listArgs := append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list hospitals: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		h, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, h)
	}
	return out, total, rows.Err()
}

// GetByID 按 id 查询。
func (r *Repo) GetByID(ctx context.Context, id int64) (*Record, error) {
	q := baseSelect + ` WHERE id = $1`
	h, err := scan(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return h, nil
}

// scan 把 pgx.Row 扫描到 Record。
func scan(row pgx.Row) (*Record, error) {
	h := &Record{}
	if err := row.Scan(
		&h.ID, &h.Name, &h.CityID, &h.Level, &h.Status, &h.Address,
		&h.Lat, &h.Lng, &h.Phone, &h.Departments, &h.Description,
		&h.CreatedAt, &h.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return h, nil
}