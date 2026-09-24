package service

import (
	"context"
	"testing"
	"time"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake ----------

type fakeRepo struct {
	sos map[int64]*SOS
	dup bool
}

func newFakeRepo() *fakeRepo { return &fakeRepo{sos: map[int64]*SOS{}} }

func (r *fakeRepo) Create(ctx context.Context, s *SOS) error {
	s.ID = int64(len(r.sos) + 1)
	r.sos[s.ID] = s
	return nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*SOS, error) {
	if s, ok := r.sos[id]; ok {
		return s, nil
	}
	return nil, ErrSOSNotFound
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	if s, ok := r.sos[id]; ok {
		s.Status = status
		return nil
	}
	return ErrSOSNotFound
}

func (r *fakeRepo) IsRecentDuplicate(ctx context.Context, orderID int64, since time.Time) (bool, error) {
	return r.dup, nil
}

type fakeLookup struct{ active bool }

func (f *fakeLookup) IsOrderActive(ctx context.Context, orderID int64) (bool, error) {
	return f.active, nil
}

type fakePub struct{ last contracts.SOSRaisedEvent }

func (p *fakePub) PublishSOSRaised(ctx context.Context, ev contracts.SOSRaisedEvent) error {
	p.last = ev
	return nil
}

func newService(repo *fakeRepo, active bool, p *fakePub) *Service {
	if repo == nil {
		repo = newFakeRepo()
	}
	return New(repo, &fakeLookup{active: active}, p)
}

// ---------- 测试 ----------

func TestRaise_OK(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	sos, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "遇到问题")
	require.NoError(t, err)
	assert.Equal(t, "raised", sos.Status)
	assert.Equal(t, int64(100), pub.last.OrderID)
}

func TestRaise_MissingIDs(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	_, err := s.Raise(context.Background(), 0, 1, 0, 0, "")
	assert.Error(t, err)
	_, err = s.Raise(context.Background(), 100, 0, 0, 0, "")
	assert.Error(t, err)
}

func TestRaise_OutOfRange(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	_, err := s.Raise(context.Background(), 100, 1, 100, 0, "")
	assert.Error(t, err)
}

func TestRaise_NoteTooLong(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'a'
	}
	_, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, string(long))
	assert.Error(t, err)
}

func TestRaise_OrderNotActive(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, false, pub)
	_, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "")
	assert.Error(t, err)
}

func TestRaise_DuplicateWithin5Min(t *testing.T) {
	pub := &fakePub{}
	r := newFakeRepo()
	r.dup = true
	s := New(r, &fakeLookup{active: true}, pub)
	_, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "")
	assert.Error(t, err)
}

func TestResolve_OK(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	sos, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "")
	require.NoError(t, err)

	require.NoError(t, s.Resolve(context.Background(), sos.ID))
}

func TestResolve_AlreadyResolved(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	sos, _ := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "")
	require.NoError(t, s.Resolve(context.Background(), sos.ID))
	assert.Error(t, s.Resolve(context.Background(), sos.ID))
}

func TestResolve_NotFound(t *testing.T) {
	pub := &fakePub{}
	s := newService(nil, true, pub)
	assert.Error(t, s.Resolve(context.Background(), 999))
}

func TestRaise_NilPublisher(t *testing.T) {
	s := New(newFakeRepo(), &fakeLookup{active: true}, nil)
	_, err := s.Raise(context.Background(), 100, 1, 39.9, 116.4, "")
	assert.NoError(t, err)
}