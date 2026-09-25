package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake ----------

type fakeRepo struct {
	byID     map[int64]*Escort
	byUserID map[int64]int64
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[int64]*Escort{}, byUserID: map[int64]int64{}}
}

func (r *fakeRepo) Create(ctx context.Context, e *Escort) error {
	r.nextID++
	e.ID = r.nextID
	if e.Status == "" {
		e.Status = "registered"
	}
	r.byID[e.ID] = e
	r.byUserID[e.UserID] = e.ID
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Escort, error) {
	if e, ok := r.byID[id]; ok {
		return e, nil
	}
	return nil, ErrEscortNotFound
}

func (r *fakeRepo) GetByUserID(ctx context.Context, userID int64) (*Escort, error) {
	id, ok := r.byUserID[userID]
	if !ok {
		return nil, ErrEscortNotFound
	}
	return r.byID[id], nil
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, s string) error {
	if e, ok := r.byID[id]; ok {
		e.Status = s
	}
	return nil
}

func (r *fakeRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	if e, ok := r.byID[id]; ok {
		e.Lat = lat
		e.Lng = lng
	}
	return nil
}

func (r *fakeRepo) UpdateCity(ctx context.Context, id int64, city string) error {
	if e, ok := r.byID[id]; ok {
		e.City = city
	}
	return nil
}

func (r *fakeRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	if e, ok := r.byID[id]; ok {
		e.AvailableFrom = from
		e.AvailableUntil = until
	}
	return nil
}

type fakePub struct {
	last AvailabilityEvent
}

func (p *fakePub) PublishAvailabilityChanged(ctx context.Context, ev AvailabilityEvent) error {
	p.last = ev
	return nil
}

type fakeQualRepo struct {
	byID map[int64]*Qualification
}

func (r *fakeQualRepo) Create(ctx context.Context, q *Qualification) error {
	q.ID = int64(len(r.byID) + 1)
	r.byID[q.ID] = q
	return nil
}
func (r *fakeQualRepo) GetByID(ctx context.Context, id, escortID int64) (*Qualification, error) {
	if q, ok := r.byID[id]; ok && q.EscortID == escortID {
		return q, nil
	}
	return nil, ErrQualificationNotFound
}
func (r *fakeQualRepo) ListByEscort(ctx context.Context, escortID int64) ([]*Qualification, error) {
	out := make([]*Qualification, 0)
	for _, q := range r.byID {
		if q.EscortID == escortID {
			out = append(out, q)
		}
	}
	return out, nil
}
func (r *fakeQualRepo) Update(ctx context.Context, q *Qualification) error {
	r.byID[q.ID] = q
	return nil
}
func (r *fakeQualRepo) Delete(ctx context.Context, id, escortID int64) error {
	if q, ok := r.byID[id]; ok && q.EscortID == escortID {
		delete(r.byID, id)
		return nil
	}
	return ErrQualificationNotFound
}

type fakeTrainingRepo struct {
	byID map[int64]*Training
}

func (r *fakeTrainingRepo) Create(ctx context.Context, t *Training) error {
	t.ID = int64(len(r.byID) + 1)
	r.byID[t.ID] = t
	return nil
}
func (r *fakeTrainingRepo) ListByEscort(ctx context.Context, escortID int64) ([]*Training, error) {
	out := make([]*Training, 0)
	for _, t := range r.byID {
		if t.EscortID == escortID {
			out = append(out, t)
		}
	}
	return out, nil
}

// ---------- 测试 ----------

func newService() (*Service, *fakeRepo, *fakePub) {
	r := newFakeRepo()
	p := &fakePub{}
	return New(r, p), r, p
}

func newServiceWithAll() (*Service, *fakeRepo, *fakePub, *fakeQualRepo, *fakeTrainingRepo) {
	r := newFakeRepo()
	p := &fakePub{}
	qr := &fakeQualRepo{byID: map[int64]*Qualification{}}
	tr := &fakeTrainingRepo{byID: map[int64]*Training{}}
	s := New(r, p).WithQualificationRepo(qr).WithTrainingRepo(tr)
	return s, r, p, qr, tr
}

func TestRegister_OK(t *testing.T) {
	s, _, _ := newService()
	e, err := s.Register(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, int64(100), e.UserID)
	assert.Equal(t, "registered", e.Status)
}

func TestRegister_Duplicate(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Register(context.Background(), 100)
	require.NoError(t, err)
	_, err = s.Register(context.Background(), 100)
	assert.Error(t, err)
}

func TestRegister_MissingUser(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Register(context.Background(), 0)
	assert.Error(t, err)
}

func TestGet_OK(t *testing.T) {
	s, _, _ := newService()
	e, _ := s.Register(context.Background(), 100)
	got, err := s.Get(context.Background(), e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
}

func TestGet_NotFound(t *testing.T) {
	s, _, _ := newService()
	_, err := s.Get(context.Background(), 999)
	assert.Error(t, err)
}

func TestGetByUserID_OK(t *testing.T) {
	s, _, _ := newService()
	_, _ = s.Register(context.Background(), 100)
	got, err := s.GetByUserID(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, int64(100), got.UserID)
}

func TestGetByUserID_NotFound(t *testing.T) {
	s, _, _ := newService()
	_, err := s.GetByUserID(context.Background(), 100)
	assert.Error(t, err)
}

func TestGetByUserID_MissingUser(t *testing.T) {
	s, _, _ := newService()
	_, err := s.GetByUserID(context.Background(), 0)
	assert.Error(t, err)
}

func TestSetAvailability_Online(t *testing.T) {
	s, r, pub := newService()
	e, _ := s.Register(context.Background(), 100)

	require.NoError(t, s.SetAvailability(context.Background(), e.ID, true, time.Now().Add(time.Hour)))
	assert.Equal(t, "available", r.byID[e.ID].Status)
	assert.True(t, pub.last.Available)
	assert.Equal(t, e.ID, pub.last.EscortID)
}

func TestSetAvailability_Offline(t *testing.T) {
	s, r, pub := newService()
	e, _ := s.Register(context.Background(), 100)

	require.NoError(t, s.SetAvailability(context.Background(), e.ID, false, time.Time{}))
	assert.Equal(t, "offline", r.byID[e.ID].Status)
	assert.False(t, pub.last.Available)
}

func TestSetAvailability_NotFound(t *testing.T) {
	s, _, _ := newService()
	err := s.SetAvailability(context.Background(), 999, true, time.Time{})
	assert.Error(t, err)
}

func TestUpdateLocation_OK(t *testing.T) {
	s, _, _ := newService()
	_, _ = s.Register(context.Background(), 100)
	require.NoError(t, s.UpdateLocation(context.Background(), 100, 39.9, 116.4))
	e, _ := s.GetByUserID(context.Background(), 100)
	assert.InDelta(t, 39.9, e.Lat, 0.001)
}

func TestUpdateLocation_NotRegistered(t *testing.T) {
	s, _, _ := newService()
	err := s.UpdateLocation(context.Background(), 999, 39.9, 116.4)
	assert.Error(t, err)
}

func TestUpdateLocation_OutOfRange(t *testing.T) {
	s, _, _ := newService()
	_, _ = s.Register(context.Background(), 100)
	err := s.UpdateLocation(context.Background(), 100, 100, 0)
	assert.Error(t, err)
}

func TestUpdateLocation_MissingUser(t *testing.T) {
	s, _, _ := newService()
	err := s.UpdateLocation(context.Background(), 0, 39.9, 116.4)
	assert.Error(t, err)
}

func TestUpdateCity_OK(t *testing.T) {
	s, r, _ := newService()
	_, _ = s.Register(context.Background(), 100)
	require.NoError(t, s.UpdateCity(context.Background(), 100, "北京"))
	assert.Equal(t, "北京", r.byID[1].City)
}

func TestUpdateCity_NotRegistered(t *testing.T) {
	s, _, _ := newService()
	err := s.UpdateCity(context.Background(), 999, "北京")
	assert.Error(t, err)
}

func TestUpdateCity_EmptyCity(t *testing.T) {
	s, _, _ := newService()
	_, _ = s.Register(context.Background(), 100)
	err := s.UpdateCity(context.Background(), 100, "  ")
	assert.Error(t, err)
}

func TestUpdateCity_MissingUser(t *testing.T) {
	s, _, _ := newService()
	err := s.UpdateCity(context.Background(), 0, "北京")
	assert.Error(t, err)
}

func TestSetAvailability_NilPublisher(t *testing.T) {
	// publisher 为 nil 也不能 panic
	s := New(newFakeRepo(), nil)
	e, _ := s.Register(context.Background(), 100)
	require.NoError(t, s.SetAvailability(context.Background(), e.ID, true, time.Time{}))
}

// ---------- Qualification ----------

func TestCreateQualification_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	q, err := s.CreateQualification(context.Background(), 100, &Qualification{
		Type:     "doctor_license",
		Number:   "DOC-001",
		IssuedAt: time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), q.EscortID) // mapped from user_id → escort.ID
	assert.False(t, q.Verified)
}

func TestCreateQualification_NoEscort(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, err := s.CreateQualification(context.Background(), 999, &Qualification{
		Type: "doctor_license", Number: "DOC-001",
	})
	assert.Error(t, err)
}

func TestCreateQualification_MissingUser(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, err := s.CreateQualification(context.Background(), 0, &Qualification{
		Type: "doctor_license", Number: "DOC-001",
	})
	assert.Error(t, err)
}

func TestCreateQualification_MissingType(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, err := s.CreateQualification(context.Background(), 100, &Qualification{
		Number: "DOC-001",
	})
	assert.Error(t, err)
}

func TestCreateQualification_MissingNumber(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, err := s.CreateQualification(context.Background(), 100, &Qualification{
		Type: "doctor_license",
	})
	assert.Error(t, err)
}

func TestCreateQualification_RepoNotConfigured(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateQualification(context.Background(), 100, &Qualification{
		Type: "doctor_license", Number: "DOC-001",
	})
	assert.Error(t, err)
}

func TestListQualifications_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, _ = s.CreateQualification(context.Background(), 100, &Qualification{Type: "doctor_license", Number: "A"})
	_, _ = s.CreateQualification(context.Background(), 100, &Qualification{Type: "nurse_license", Number: "B"})
	list, err := s.ListQualifications(context.Background(), 100)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListQualifications_NotRegistered(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, err := s.ListQualifications(context.Background(), 999)
	assert.Error(t, err)
}

func TestListQualifications_RepoNotConfigured(t *testing.T) {
	s, _, _ := newService()
	_, err := s.ListQualifications(context.Background(), 100)
	assert.Error(t, err)
}

func TestUpdateQualification_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	q, _ := s.CreateQualification(context.Background(), 100, &Qualification{Type: "doctor_license", Number: "A"})
	out, err := s.UpdateQualification(context.Background(), 100, q.ID, map[string]any{
		"number": "A-new",
	})
	require.NoError(t, err)
	assert.Equal(t, "A-new", out.Number)
}

func TestUpdateQualification_NotFound(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, err := s.UpdateQualification(context.Background(), 100, 999, map[string]any{"number": "X"})
	assert.Error(t, err)
}

func TestUpdateQualification_MissingID(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, err := s.UpdateQualification(context.Background(), 100, 0, map[string]any{"number": "X"})
	assert.Error(t, err)
}

func TestDeleteQualification_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	q, _ := s.CreateQualification(context.Background(), 100, &Qualification{Type: "doctor_license", Number: "A"})
	require.NoError(t, s.DeleteQualification(context.Background(), 100, q.ID))
	list, _ := s.ListQualifications(context.Background(), 100)
	assert.Len(t, list, 0)
}

func TestDeleteQualification_NotFound(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	err := s.DeleteQualification(context.Background(), 100, 999)
	assert.Error(t, err)
}

// ---------- Training ----------

func TestCreateTraining_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	t2, err := s.CreateTraining(context.Background(), 100, &Training{
		Title: "急救培训", Provider: "红十字会", CompletedAt: time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, "急救培训", t2.Title)
}

func TestCreateTraining_NoEscort(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, err := s.CreateTraining(context.Background(), 999, &Training{Title: "x"})
	assert.Error(t, err)
}

func TestCreateTraining_MissingTitle(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, err := s.CreateTraining(context.Background(), 100, &Training{})
	assert.Error(t, err)
}

func TestCreateTraining_RepoNotConfigured(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateTraining(context.Background(), 100, &Training{Title: "x"})
	assert.Error(t, err)
}

func TestListTrainings_OK(t *testing.T) {
	s, _, _, _, _ := newServiceWithAll()
	_, _ = s.Register(context.Background(), 100)
	_, _ = s.CreateTraining(context.Background(), 100, &Training{Title: "A"})
	_, _ = s.CreateTraining(context.Background(), 100, &Training{Title: "B"})
	list, err := s.ListTrainings(context.Background(), 100)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListTrainings_RepoNotConfigured(t *testing.T) {
	s, _, _ := newService()
	_, err := s.ListTrainings(context.Background(), 100)
	assert.Error(t, err)
}

// 兜底编译。
var _ = errors.New
