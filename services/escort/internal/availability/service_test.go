package availability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo 提供 fake 仓储（实现 AvailabilityRepo 接口）。
type fakeRepo struct {
	rows   map[int64]*Availability
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{rows: map[int64]*Availability{}}
}

func (r *fakeRepo) Create(ctx context.Context, escortID int64, start, end time.Time) (*Availability, error) {
	for _, a := range r.rows {
		if a.EscortID == escortID && a.Status != StatusCanceled && a.Overlaps(&Availability{StartAt: start, EndAt: end}) {
			return nil, ErrAvailabilityConflict
		}
	}
	r.nextID++
	a := &Availability{ID: r.nextID, EscortID: escortID, StartAt: start, EndAt: end, Status: StatusAvailable}
	r.rows[a.ID] = a
	return a, nil
}

func (r *fakeRepo) FindByID(ctx context.Context, id int64) (*Availability, error) {
	a, ok := r.rows[id]
	if !ok {
		return nil, ErrAvailabilityNotFound
	}
	return a, nil
}

func (r *fakeRepo) Delete(ctx context.Context, id int64, escortID int64) error {
	a, ok := r.rows[id]
	if !ok {
		return ErrAvailabilityNotFound
	}
	if a.EscortID != escortID {
		return ErrNotOwner
	}
	if a.Status == StatusBooked {
		return ErrBookedAlready
	}
	delete(r.rows, id)
	return nil
}

func (r *fakeRepo) ListByEscort(ctx context.Context, escortID int64) ([]*Availability, error) {
	out := make([]*Availability, 0)
	for _, a := range r.rows {
		if a.EscortID == escortID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListAvailableByTime(ctx context.Context, start, end time.Time, limit int) ([]*Availability, error) {
	out := make([]*Availability, 0)
	for _, a := range r.rows {
		if a.Status == StatusAvailable && a.StartAt.Before(end) && start.Before(a.EndAt) {
			out = append(out, a)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *fakeRepo) BookByOrder(ctx context.Context, id int64, orderID int64) error {
	a, ok := r.rows[id]
	if !ok {
		return ErrAvailabilityNotFound
	}
	if a.Status != StatusAvailable {
		return ErrNotBookable
	}
	a.Status = StatusBooked
	oid := orderID
	a.BookedOrderID = &oid
	return nil
}

func (r *fakeRepo) ReleaseByOrder(ctx context.Context, orderID int64) error {
	for _, a := range r.rows {
		if a.BookedOrderID != nil && *a.BookedOrderID == orderID {
			a.Status = StatusAvailable
			a.BookedOrderID = nil
			return nil
		}
	}
	return ErrAvailabilityNotFound
}

// fixedClock 用于测试注入时间。
type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func newServiceWithClock(r *fakeRepo, now time.Time) *Service {
	return NewService(r).WithClock(fixedClock{t: now})
}

// TestAddAvailability_OK 验证陪诊师加时段成功。
func TestAddAvailability_OK(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	start := now.Add(2 * time.Hour)
	end := now.Add(5 * time.Hour)
	a, err := s.AddAvailability(context.Background(), 7, 7, start, end)
	require.NoError(t, err)
	assert.Equal(t, int64(7), a.EscortID)
	assert.Equal(t, StatusAvailable, a.Status)
	assert.Equal(t, start, a.StartAt)
}

// TestAddAvailability_NotOwner 验证非 owner 加时段被拒。
func TestAddAvailability_NotOwner(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	_, err := s.AddAvailability(context.Background(), 7, 99, now.Add(time.Hour), now.Add(2*time.Hour))
	assert.ErrorIs(t, err, ErrNotOwner)
}

// TestAddAvailability_TimeInvalid 验证时间非法（end <= start）被拒。
func TestAddAvailability_TimeInvalid(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	_, err := s.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(time.Hour))
	assert.ErrorIs(t, err, ErrTimeInvalid)
}

// TestAddAvailability_InPast 验证过去时段被拒。
func TestAddAvailability_InPast(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	_, err := s.AddAvailability(context.Background(), 7, 7, now.Add(-2*time.Hour), now.Add(time.Hour))
	assert.ErrorIs(t, err, ErrTimeInvalid)
}

// TestAddAvailability_Conflict 验证时段冲突被拒。
func TestAddAvailability_Conflict(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	start := now.Add(2 * time.Hour)
	end := now.Add(5 * time.Hour)
	_, err := s.AddAvailability(context.Background(), 7, 7, start, end)
	require.NoError(t, err)
	_, err = s.AddAvailability(context.Background(), 7, 7, start.Add(30*time.Minute), end.Add(30*time.Minute))
	assert.ErrorIs(t, err, ErrAvailabilityConflict)
}

// TestRemoveAvailability_OK 验证删除自己的时段成功。
func TestRemoveAvailability_OK(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	start := now.Add(2 * time.Hour)
	end := now.Add(5 * time.Hour)
	a, _ := s.AddAvailability(context.Background(), 7, 7, start, end)
	require.NoError(t, s.RemoveAvailability(context.Background(), a.ID, 7))
	err := s.RemoveAvailability(context.Background(), a.ID, 7)
	assert.True(t, errors.Is(err, ErrAvailabilityNotFound) || errors.Is(err, ErrNotOwner))
}

// TestRemoveAvailability_BookedRejected 验证 booked 时段不可删。
func TestRemoveAvailability_BookedRejected(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	a, _ := s.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(2*time.Hour))
	require.NoError(t, s.BookForOrder(context.Background(), a.ID, 100))
	err := s.RemoveAvailability(context.Background(), a.ID, 7)
	assert.ErrorIs(t, err, ErrBookedAlready)
}

// TestHasAvailabilityFor_OK 验证陪诊师在时段窗口内有可用时段。
func TestHasAvailabilityFor_OK(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	_, _ = s.AddAvailability(context.Background(), 7, 7, now.Add(3*time.Hour), now.Add(5*time.Hour))
	ok, err := s.HasAvailabilityFor(context.Background(), 7, now.Add(4*time.Hour))
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestHasAvailabilityFor_False 验证陪诊师无时段返回 false。
func TestHasAvailabilityFor_False(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	ok, err := s.HasAvailabilityFor(context.Background(), 7, now.Add(48*time.Hour))
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestBookAndReleaseForOrder 验证 book → release 循环。
func TestBookAndReleaseForOrder(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newServiceWithClock(newFakeRepo(), now)
	a, _ := s.AddAvailability(context.Background(), 7, 7, now.Add(time.Hour), now.Add(2*time.Hour))
	require.NoError(t, s.BookForOrder(context.Background(), a.ID, 100))
	got, _ := s.repo.FindByID(context.Background(), a.ID)
	assert.Equal(t, StatusBooked, got.Status)
	require.NotNil(t, got.BookedOrderID)
	assert.Equal(t, int64(100), *got.BookedOrderID)

	require.NoError(t, s.ReleaseForOrder(context.Background(), 100))
	got, _ = s.repo.FindByID(context.Background(), a.ID)
	assert.Equal(t, StatusAvailable, got.Status)
	assert.Nil(t, got.BookedOrderID)
}