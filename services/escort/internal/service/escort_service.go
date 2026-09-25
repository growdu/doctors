// Package service 是 escort-service 的业务编排层。
//
// 设计要点：
//   - 陪诊师实体：用户（patient/escort）的 escort 子集；通过 user_id 关联。
//   - 可服务区间（available_from / available_until）用于评分过滤。
//   - SetAvailability 触发 events.EscortAvailable / EscortUnavailable；
//     match-service 消费后更新候选池。
//   - Status: "registered" | "available" | "busy" | "offline"。
//   - qualifications：陪诊师资质（医师执业证、护士证等）；支持 CRUD。
//   - trainings：培训记录；支持新增 + 列表。
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// Escort 是陪诊师的最小业务视图。
type Escort struct {
	ID             int64
	UserID         int64
	City           string
	Lat            float64
	Lng            float64
	Rating         float64
	Status         string
	AvailableFrom  time.Time
	AvailableUntil time.Time
	CreatedAt      time.Time
}

// EscortRepo 是仓储契约。
type EscortRepo interface {
	Create(ctx context.Context, e *Escort) error
	GetByID(ctx context.Context, id int64) (*Escort, error)
	GetByUserID(ctx context.Context, userID int64) (*Escort, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateLocation(ctx context.Context, id int64, lat, lng float64) error
	UpdateCity(ctx context.Context, id int64, city string) error
	UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error
}

// AvailabilityEvent 是发布给 match-service 的最小事件。
type AvailabilityEvent struct {
	EscortID   int64     `json:"escort_id"`
	UserID     int64     `json:"user_id"`
	City       string    `json:"city"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Available  bool      `json:"available"`
	ValidUntil time.Time `json:"valid_until"`
}

// Publisher 是事件发布抽象；match-service 消费后调整候选池。
type Publisher interface {
	PublishAvailabilityChanged(ctx context.Context, ev AvailabilityEvent) error
}

// Qualification 是陪诊师资质记录。
type Qualification struct {
	ID         int64
	EscortID   int64
	Type       string // "doctor_license" | "nurse_license" | "pharmacist" | "other"
	Number     string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	ImageURL   string
	Verified   bool
	CreatedAt  time.Time
}

// QualificationRepo 是资质的仓储契约。
type QualificationRepo interface {
	Create(ctx context.Context, q *Qualification) error
	GetByID(ctx context.Context, id, escortID int64) (*Qualification, error)
	ListByEscort(ctx context.Context, escortID int64) ([]*Qualification, error)
	Update(ctx context.Context, q *Qualification) error
	Delete(ctx context.Context, id, escortID int64) error
}

// Training 是培训记录。
type Training struct {
	ID          int64
	EscortID    int64
	Title       string
	Provider    string
	CompletedAt time.Time
	ExpiresAt   time.Time
	CertificateURL string
	CreatedAt   time.Time
}

// TrainingRepo 是培训的仓储契约。
type TrainingRepo interface {
	Create(ctx context.Context, t *Training) error
	ListByEscort(ctx context.Context, escortID int64) ([]*Training, error)
}

// ErrEscortNotFound 查无结果。
var ErrEscortNotFound = errors.New("escort service: not found")
var ErrQualificationNotFound = errors.New("escort service: qualification not found")

// Service 是 escort 业务编排器。
type Service struct {
	escortRepo       EscortRepo
	qualificationRepo QualificationRepo
	trainingRepo      TrainingRepo
	publisher         Publisher
}

// New 装配 Service。
func New(r EscortRepo, p Publisher) *Service {
	return &Service{escortRepo: r, publisher: p}
}

// WithQualificationRepo 注入 qualification repo（业务接口独立，便于单测 fake）。
func (s *Service) WithQualificationRepo(r QualificationRepo) *Service {
	s.qualificationRepo = r
	return s
}

// WithTrainingRepo 注入 training repo。
func (s *Service) WithTrainingRepo(r TrainingRepo) *Service {
	s.trainingRepo = r
	return s
}

// Register 注册一个新陪诊师；默认 status="registered"。
//   - userID 必须 > 0 且在 users 表存在（v1 不验，交给上层）
//   - 默认城市为空，location 由后续 PATCH 设置
func (s *Service) Register(ctx context.Context, userID int64) (*Escort, error) {
	if userID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "user_id required")
	}
	if existing, _ := s.escortRepo.GetByUserID(ctx, userID); existing != nil {
		return nil, errs.New(errs.CodeConflict, "user already registered as escort")
	}
	e := &Escort{
		UserID: userID,
		Status: "registered",
	}
	if err := s.escortRepo.Create(ctx, e); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create escort", err)
	}
	return e, nil
}

// Get 取一个 escort 详情。
func (s *Service) Get(ctx context.Context, id int64) (*Escort, error) {
	e, err := s.escortRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get escort", err)
	}
	return e, nil
}

// GetByUserID 通过 user_id 取 escort（用于 /me 路径）。
func (s *Service) GetByUserID(ctx context.Context, userID int64) (*Escort, error) {
	if userID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "user_id required")
	}
	e, err := s.escortRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get escort by user", err)
	}
	return e, nil
}

// SetAvailability 设置可服务状态；触发事件发布。
//   - available=true：status="available"，发布 EscortAvailable
//   - available=false：status="offline"，发布 EscortUnavailable
func (s *Service) SetAvailability(ctx context.Context, id int64, available bool, validUntil time.Time) error {
	e, err := s.escortRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not found")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	status := "offline"
	if available {
		status = "available"
	}
	if err := s.escortRepo.UpdateStatus(ctx, id, status); err != nil {
		return errs.Wrap(errs.CodeInternal, "update status", err)
	}
	if available && validUntil.IsZero() {
		validUntil = time.Now().Add(2 * time.Hour)
	}
	if err := s.escortRepo.UpdateAvailability(ctx, id, time.Now(), validUntil); err != nil {
		return errs.Wrap(errs.CodeInternal, "update availability", err)
	}

	// 发布事件（失败不阻塞业务；log 错误即可）
	if s.publisher != nil {
		if err := s.publisher.PublishAvailabilityChanged(ctx, AvailabilityEvent{
			EscortID:   id,
			UserID:     e.UserID,
			City:       e.City,
			Lat:        e.Lat,
			Lng:        e.Lng,
			Available:  available,
			ValidUntil: validUntil,
		}); err != nil {
			// publish 失败仅记日志，不影响主流程
			_ = err
		}
	}
	return nil
}

// UpdateLocation 更新经纬度（/me/location）。
//
// callerID = JWT user_id；通过 GetByUserID 找 escort 记录，确保是 owner。
func (s *Service) UpdateLocation(ctx context.Context, callerID int64, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return errs.New(errs.CodeParamInvalid, "lat/lng out of range")
	}
	if callerID == 0 {
		return errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not registered")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if err := s.escortRepo.UpdateLocation(ctx, e.ID, lat, lng); err != nil {
		return errs.Wrap(errs.CodeInternal, "update location", err)
	}
	return nil
}

// UpdateCity 修改服务城市（/me/city）。
//
// callerID = JWT user_id；通过 GetByUserID 找 escort 记录，确保是 owner。
func (s *Service) UpdateCity(ctx context.Context, callerID int64, city string) error {
	city = strings.TrimSpace(city)
	if city == "" {
		return errs.New(errs.CodeParamInvalid, "city required")
	}
	if callerID == 0 {
		return errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not registered")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if err := s.escortRepo.UpdateCity(ctx, e.ID, city); err != nil {
		return errs.Wrap(errs.CodeInternal, "update city", err)
	}
	return nil
}

// ---------- Qualification ----------

// CreateQualification 陪诊师新增资质。
func (s *Service) CreateQualification(ctx context.Context, callerID int64, q *Qualification) (*Qualification, error) {
	if s.qualificationRepo == nil {
		return nil, errs.New(errs.CodeUnavailable, "qualification repo not configured")
	}
	if callerID == 0 {
		return nil, errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not registered")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if strings.TrimSpace(q.Type) == "" {
		return nil, errs.New(errs.CodeParamInvalid, "type required")
	}
	if strings.TrimSpace(q.Number) == "" {
		return nil, errs.New(errs.CodeParamInvalid, "number required")
	}
	q.EscortID = e.ID
	q.Verified = false
	if err := s.qualificationRepo.Create(ctx, q); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create qualification", err)
	}
	return q, nil
}

// ListQualifications 陪诊师查自己的资质列表。
func (s *Service) ListQualifications(ctx context.Context, callerID int64) ([]*Qualification, error) {
	if s.qualificationRepo == nil {
		return nil, errs.New(errs.CodeUnavailable, "qualification repo not configured")
	}
	if callerID == 0 {
		return nil, errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not registered")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	return s.qualificationRepo.ListByEscort(ctx, e.ID)
}

// UpdateQualification 陪诊师改资质。
func (s *Service) UpdateQualification(ctx context.Context, callerID, qid int64, patch map[string]any) (*Qualification, error) {
	if s.qualificationRepo == nil {
		return nil, errs.New(errs.CodeUnavailable, "qualification repo not configured")
	}
	if callerID == 0 || qid == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "id required")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not registered")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	q, err := s.qualificationRepo.GetByID(ctx, qid, e.ID)
	if err != nil {
		if errors.Is(err, ErrQualificationNotFound) {
			return nil, errs.New(errs.CodeNotFound, "qualification not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find qualification", err)
	}
	if v, ok := patch["type"].(string); ok && strings.TrimSpace(v) != "" {
		q.Type = v
	}
	if v, ok := patch["number"].(string); ok && strings.TrimSpace(v) != "" {
		q.Number = v
	}
	if v, ok := patch["image_url"].(string); ok {
		q.ImageURL = v
	}
	if v, ok := patch["expires_at"].(time.Time); ok {
		q.ExpiresAt = v
	}
	if err := s.qualificationRepo.Update(ctx, q); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "update qualification", err)
	}
	return q, nil
}

// DeleteQualification 陪诊师删资质。
func (s *Service) DeleteQualification(ctx context.Context, callerID, qid int64) error {
	if s.qualificationRepo == nil {
		return errs.New(errs.CodeUnavailable, "qualification repo not configured")
	}
	if callerID == 0 || qid == 0 {
		return errs.New(errs.CodeParamInvalid, "id required")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return errs.New(errs.CodeNotFound, "escort not registered")
		}
		return errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if err := s.qualificationRepo.Delete(ctx, qid, e.ID); err != nil {
		if errors.Is(err, ErrQualificationNotFound) {
			return errs.New(errs.CodeNotFound, "qualification not found")
		}
		return errs.Wrap(errs.CodeInternal, "delete qualification", err)
	}
	return nil
}

// ---------- Training ----------

// CreateTraining 陪诊师新增培训记录。
func (s *Service) CreateTraining(ctx context.Context, callerID int64, t *Training) (*Training, error) {
	if s.trainingRepo == nil {
		return nil, errs.New(errs.CodeUnavailable, "training repo not configured")
	}
	if callerID == 0 {
		return nil, errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not registered")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	if strings.TrimSpace(t.Title) == "" {
		return nil, errs.New(errs.CodeParamInvalid, "title required")
	}
	t.EscortID = e.ID
	if err := s.trainingRepo.Create(ctx, t); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create training", err)
	}
	return t, nil
}

// ListTrainings 陪诊师查自己的培训记录。
func (s *Service) ListTrainings(ctx context.Context, callerID int64) ([]*Training, error) {
	if s.trainingRepo == nil {
		return nil, errs.New(errs.CodeUnavailable, "training repo not configured")
	}
	if callerID == 0 {
		return nil, errs.New(errs.CodeUnauthorized, "no user")
	}
	e, err := s.escortRepo.GetByUserID(ctx, callerID)
	if err != nil {
		if errors.Is(err, ErrEscortNotFound) {
			return nil, errs.New(errs.CodeNotFound, "escort not registered")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find escort", err)
	}
	return s.trainingRepo.ListByEscort(ctx, e.ID)
}
