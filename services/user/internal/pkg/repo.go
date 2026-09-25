// Package pkg 实现 user-service 服务包的数据访问层。
//
// 设计要点：
//   - 服务包挂在医院下（FK: hospital_id → hospitals.id）。
//   - v1 只读 + 简单查询：ListByHospital + GetByID。
//   - 无越权问题（公共资源）。
//   - price 用 float64 表达；DB 存 NUMERIC(10,2)；handler 端转 string 防 JS 精度漂移。
package pkg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Record 映射 packages 表行。
type Record struct {
	ID          int64
	HospitalID  int64
	Name        string
	Type        string  // half_day | full_day | single_item
	DurationMin int
	Price       float64
	Status      string  // active | inactive
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ErrNotFound 是查询无结果哨兵。
var ErrNotFound = errors.New("pkg: not found")

// Repo 是 packages 表的仓储。
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo 构造仓储。
func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const baseSelect = `
	SELECT id, hospital_id, name, type, duration_min, price, status,
	       description, created_at, updated_at
	FROM packages`

// ListByHospital 列出某医院的服务包（仅 active）。
func (r *Repo) ListByHospital(ctx context.Context, hospitalID int64) ([]*Record, error) {
	q := baseSelect + ` WHERE hospital_id = $1 AND status = 'active' ORDER BY price ASC, id ASC`
	rows, err := r.pool.Query(ctx, q, hospitalID)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	defer rows.Close()
	out := []*Record{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetByID 按 id 查询。
func (r *Repo) GetByID(ctx context.Context, id int64) (*Record, error) {
	q := baseSelect + ` WHERE id = $1`
	p, err := scan(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

// scan 把 pgx.Row 扫描到 Record。
func scan(row pgx.Row) (*Record, error) {
	p := &Record{}
	if err := row.Scan(
		&p.ID, &p.HospitalID, &p.Name, &p.Type, &p.DurationMin, &p.Price, &p.Status,
		&p.Description, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}