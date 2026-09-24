package service

import (
	"context"
	"testing"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake ----------

type fakeRepo struct {
	reviews  map[int64]*Review
	byOrder  map[int64]int64
	nextID   int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{reviews: map[int64]*Review{}, byOrder: map[int64]int64{}}
}

func (r *fakeRepo) Create(ctx context.Context, rv *Review) error {
	r.nextID++
	rv.ID = r.nextID
	r.reviews[rv.ID] = rv
	r.byOrder[rv.OrderID] = rv.ID
	return nil
}

func (r *fakeRepo) GetByOrderID(ctx context.Context, orderID int64) (*Review, error) {
	id, ok := r.byOrder[orderID]
	if !ok {
		return nil, ErrReviewNotFound
	}
	return r.reviews[id], nil
}

func (r *fakeRepo) ListByEscort(ctx context.Context, escortID int64, limit, offset int) ([]*Review, error) {
	out := make([]*Review, 0)
	for _, rv := range r.reviews {
		if rv.EscortID == escortID {
			out = append(out, rv)
		}
	}
	return out, nil
}

type fakePub struct {
	last contracts.OrderReviewedEvent
}

func (p *fakePub) PublishOrderReviewed(ctx context.Context, ev contracts.OrderReviewedEvent) error {
	p.last = ev
	return nil
}

// ---------- 测试 ----------

func newService() (*Service, *fakeRepo, *fakePub) {
	r := newFakeRepo()
	p := &fakePub{}
	return New(r, p), r, p
}

func TestCreateReview_OK(t *testing.T) {
	s, _, pub := newService()
	r, err := s.CreateReview(context.Background(), 1, 100, 7, 5, "非常专业！")
	require.NoError(t, err)
	assert.Equal(t, 5, r.Rating)
	assert.Equal(t, int64(100), r.OrderID)
	assert.Equal(t, int64(100), pub.last.OrderID)
	assert.Equal(t, 5, pub.last.Rating)
}

func TestCreateReview_BadRating(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateReview(context.Background(), 1, 100, 7, 0, "")
	assert.Error(t, err)
	_, err = s.CreateReview(context.Background(), 1, 100, 7, 6, "")
	assert.Error(t, err)
}

func TestCreateReview_MissingIDs(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateReview(context.Background(), 0, 100, 7, 5, "")
	assert.Error(t, err)
	_, err = s.CreateReview(context.Background(), 1, 0, 7, 5, "")
	assert.Error(t, err)
	_, err = s.CreateReview(context.Background(), 1, 100, 0, 5, "")
	assert.Error(t, err)
}

func TestCreateReview_Duplicate(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateReview(context.Background(), 1, 100, 7, 5, "好")
	require.NoError(t, err)
	_, err = s.CreateReview(context.Background(), 1, 100, 7, 3, "一般")
	assert.Error(t, err)
}

func TestCreateReview_CommentTooLong(t *testing.T) {
	s, _, _ := newService()
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'a'
	}
	_, err := s.CreateReview(context.Background(), 1, 100, 7, 5, string(long))
	assert.Error(t, err)
}

func TestGetByOrder_OK(t *testing.T) {
	s, _, _ := newService()
	_, err := s.CreateReview(context.Background(), 1, 100, 7, 5, "好")
	require.NoError(t, err)
	got, err := s.GetByOrder(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, 5, got.Rating)
}

func TestGetByOrder_NotFound(t *testing.T) {
	s, _, _ := newService()
	_, err := s.GetByOrder(context.Background(), 999)
	assert.Error(t, err)
}

func TestListByEscort(t *testing.T) {
	s, _, _ := newService()
	for i := 0; i < 3; i++ {
		_, _ = s.CreateReview(context.Background(), 1, int64(100+i), 7, 5, "好")
	}
	_, _ = s.CreateReview(context.Background(), 1, 200, 8, 4, "好") // 不同 escort
	list, err := s.ListByEscort(context.Background(), 7, 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestCreateReview_NilPublisher(t *testing.T) {
	s := New(newFakeRepo(), nil)
	_, err := s.CreateReview(context.Background(), 1, 100, 7, 5, "好")
	assert.NoError(t, err)
}