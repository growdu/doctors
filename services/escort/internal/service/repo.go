// Package service - escort-service 仓储层（v1 in-memory 简化实现）。
//
// 设计要点：
//   - v1 不接真实 PG：内存版 escortProfileRepo / qualificationRepo / trainingRepo。
//   - 重启清空（v1 范围；prod 应替换为 pgx 直写）。
//   - 之所以保留 New*Repo(pool) 签名：
//   - 1）符合 §31 "11 服务真实 PG 接入" 接口约定（pool 参数透传，便于 v2 替换）；
//   - 2）调用方现在传 nil 也能跑；未来接 DB 时只换实现，不改 main。
package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrNotFound 是所有 escort 内 in-memory repo 的通用 sentinel。
//
//	为避免与 ErrEscortNotFound / ErrQualificationNotFound 冲突，新加：
//	ErrProfileNotFound / ErrQualTrainingNotFound。
var ErrProfileNotFound = errors.New("escort service: profile not found")
var ErrQualTrainingNotFound = errors.New("escort service: qualification/training not found")

// ---------- ProfileRepo（escort_profile 表）----------

// ProfileRepo 是 escort_profile 表的仓储契约（v1 in-memory）。
type ProfileRepo struct {
	mu      sync.RWMutex
	byID    map[int64]*Escort
	byUser  map[int64]int64 // userID -> escortID
	nextID  int64
}

// NewProfileRepo 构造内存版 escort_profile 仓储。
//
//	pool 参数保留以兼容 §31 接口约定；当前实现不依赖 pgxpool。
func NewProfileRepo(pool any) *ProfileRepo {
	return &ProfileRepo{
		byID:   make(map[int64]*Escort),
		byUser: make(map[int64]int64),
	}
}

// Create 写入新 escort 记录。
func (r *ProfileRepo) Create(ctx context.Context, e *Escort) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.ID == 0 {
		e.ID = atomic.AddInt64(&r.nextID, 1)
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	r.byID[e.ID] = e
	r.byUser[e.UserID] = e.ID
	return nil
}

// GetByID 按主键查找。
func (r *ProfileRepo) GetByID(ctx context.Context, id int64) (*Escort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byID[id]
	if !ok {
		return nil, ErrProfileNotFound
	}
	return e, nil
}

// GetByUserID 按 userID 查找。
func (r *ProfileRepo) GetByUserID(ctx context.Context, userID int64) (*Escort, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byUser[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	return r.byID[id], nil
}

// UpdateStatus 更新状态。
func (r *ProfileRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.byID[id]
	if !ok {
		return ErrProfileNotFound
	}
	e.Status = status
	return nil
}

// UpdateLocation 更新经纬度。
func (r *ProfileRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.byID[id]
	if !ok {
		return ErrProfileNotFound
	}
	e.Lat = lat
	e.Lng = lng
	return nil
}

// UpdateCity 更新服务城市。
func (r *ProfileRepo) UpdateCity(ctx context.Context, id int64, city string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.byID[id]
	if !ok {
		return ErrProfileNotFound
	}
	e.City = city
	return nil
}

// UpdateAvailability 更新可服务时间窗。
func (r *ProfileRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.byID[id]
	if !ok {
		return ErrProfileNotFound
	}
	e.AvailableFrom = from
	e.AvailableUntil = until
	return nil
}

// ---------- QualificationRepo（qualifications 表）----------

// MemoryQualificationRepo 是 qualifications 表的仓储实现（v1 in-memory）。
//
//	用 MemoryXxxRepo 命名以避免与 escort_service.go 的接口名 QualificationRepo 冲突。
type MemoryQualificationRepo struct {
	mu     sync.RWMutex
	byID   map[int64]*Qualification
	byEsc  map[int64][]int64 // escortID -> []qualID
	nextID int64
}

// NewQualificationRepo 构造内存版 qualifications 仓储。
//
//	pool 参数保留以兼容 §31 接口约定；当前实现不依赖 pgxpool。
func NewQualificationRepo(pool any) *MemoryQualificationRepo {
	return &MemoryQualificationRepo{
		byID:  make(map[int64]*Qualification),
		byEsc: make(map[int64][]int64),
	}
}

// Create 写入新资质记录。
func (r *MemoryQualificationRepo) Create(ctx context.Context, q *Qualification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	q.ID = atomic.AddInt64(&r.nextID, 1)
	if q.CreatedAt.IsZero() {
		q.CreatedAt = time.Now()
	}
	r.byID[q.ID] = q
	r.byEsc[q.EscortID] = append(r.byEsc[q.EscortID], q.ID)
	return nil
}

// GetByID 按 (id, escortID) 取；校验 owner。
func (r *MemoryQualificationRepo) GetByID(ctx context.Context, id, escortID int64) (*Qualification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q, ok := r.byID[id]
	if !ok || q.EscortID != escortID {
		return nil, ErrQualTrainingNotFound
	}
	return q, nil
}

// ListByEscort 返回某 escort 的全部资质。
func (r *MemoryQualificationRepo) ListByEscort(ctx context.Context, escortID int64) ([]*Qualification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byEsc[escortID]
	out := make([]*Qualification, 0, len(ids))
	for _, id := range ids {
		if q, ok := r.byID[id]; ok {
			out = append(out, q)
		}
	}
	return out, nil
}

// Update 更新资质内容。
func (r *MemoryQualificationRepo) Update(ctx context.Context, q *Qualification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[q.ID]; !ok {
		return ErrQualTrainingNotFound
	}
	r.byID[q.ID] = q
	return nil
}

// Delete 删除资质（仅 owner）。
func (r *MemoryQualificationRepo) Delete(ctx context.Context, id, escortID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, ok := r.byID[id]
	if !ok || q.EscortID != escortID {
		return ErrQualTrainingNotFound
	}
	delete(r.byID, id)
	r.byEsc[escortID] = removeID(r.byEsc[escortID], id)
	return nil
}

// removeID 从 slice 删除一个 id（保留顺序）。
func removeID(s []int64, id int64) []int64 {
	for i, v := range s {
		if v == id {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// ---------- TrainingRepo（trainings 表）----------

// MemoryTrainingRepo 是 trainings 表的仓储实现（v1 in-memory）。
type MemoryTrainingRepo struct {
	mu     sync.RWMutex
	byID   map[int64]*Training
	byEsc  map[int64][]int64
	nextID int64
}

// NewTrainingRepo 构造内存版 trainings 仓储。
func NewTrainingRepo(pool any) *MemoryTrainingRepo {
	return &MemoryTrainingRepo{
		byID:  make(map[int64]*Training),
		byEsc: make(map[int64][]int64),
	}
}

// Create 写入新培训记录。
func (r *MemoryTrainingRepo) Create(ctx context.Context, t *Training) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID = atomic.AddInt64(&r.nextID, 1)
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	r.byID[t.ID] = t
	r.byEsc[t.EscortID] = append(r.byEsc[t.EscortID], t.ID)
	return nil
}

// ListByEscort 返回某 escort 的全部培训记录。
func (r *MemoryTrainingRepo) ListByEscort(ctx context.Context, escortID int64) ([]*Training, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byEsc[escortID]
	out := make([]*Training, 0, len(ids))
	for _, id := range ids {
		if t, ok := r.byID[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}