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

func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	if e, ok := r.byID[id]; ok {
		e.Status = status
		return nil
	}
	return ErrEscortNotFound
}

func (r *fakeRepo) UpdateLocation(ctx context.Context, id int64, lat, lng float64) error {
	if e, ok := r.byID[id]; ok {
		e.Lat = lat
		e.Lng = lng
		return nil
	}
	return ErrEscortNotFound
}

func (r *fakeRepo) UpdateAvailability(ctx context.Context, id int64, from, until time.Time) error {
	if e, ok := r.byID[id]; ok {
		e.AvailableFrom = from
		e.AvailableUntil = until
		return nil
	}
	return ErrEscortNotFound
}

type fakePub struct {
	last AvailabilityEvent
}

func (p *fakePub) PublishAvailabilityChanged(ctx context.Context, ev AvailabilityEvent) error {
	p.last = ev
	return nil
}

// ---------- 测试 ----------

func newService() (*Service, *fakeRepo, *fakePub) {
	r := newFakeRepo()
	p := &fakePub{}
	return New(r, p), r, p
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
	e, _ := s.Register(context.Background(), 100)
	require.NoError(t, s.UpdateLocation(context.Background(), e.ID, 100, 39.9, 116.4))
	got, _ := s.Get(context.Background(), e.ID)
	assert.InDelta(t, 39.9, got.Lat, 0.001)
}

func TestUpdateLocation_NotSelf(t *testing.T) {
	s, _, _ := newService()
	e, _ := s.Register(context.Background(), 100)
	err := s.UpdateLocation(context.Background(), e.ID, 999, 39.9, 116.4)
	assert.Error(t, err)
}

func TestUpdateLocation_OutOfRange(t *testing.T) {
	s, _, _ := newService()
	e, _ := s.Register(context.Background(), 100)
	err := s.UpdateLocation(context.Background(), e.ID, 100, 100, 0)
	assert.Error(t, err)
}

func TestSetAvailability_NilPublisher(t *testing.T) {
	// publisher 为 nil 也不能 panic
	s := New(newFakeRepo(), nil)
	e, _ := s.Register(context.Background(), 100)
	require.NoError(t, s.SetAvailability(context.Background(), e.ID, true, time.Time{}))
}

// 兜底编译。
var _ = errors.New