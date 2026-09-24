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
	msgs map[int64]*Message
}

func (r *fakeRepo) CreateConversation(ctx context.Context, c *Conversation) error { return nil }
func (r *fakeRepo) CreateMessage(ctx context.Context, m *Message) error {
	if m.ID == 0 {
		m.ID = int64(len(r.msgs) + 1)
	}
	r.msgs[m.ID] = m
	return nil
}
func (r *fakeRepo) ListMessagesByOrder(ctx context.Context, orderID int64, limit, offset int) ([]*Message, error) {
	out := make([]*Message, 0)
	for _, m := range r.msgs {
		if m.OrderID == orderID {
			out = append(out, m)
		}
	}
	return out, nil
}

type fakePub struct {
	last contracts.MessageSentEvent
}

func (p *fakePub) PublishMessageSent(ctx context.Context, ev contracts.MessageSentEvent) error {
	p.last = ev
	return nil
}

// ---------- 测试 ----------

func newService() (*Service, *fakeRepo, *fakePub) {
	r := &fakeRepo{msgs: map[int64]*Message{}}
	p := &fakePub{}
	return New(r, p), r, p
}

func TestSendMessage_OK(t *testing.T) {
	s, _, pub := newService()
	m, err := s.SendMessage(context.Background(), 100, 1, 2, "你好")
	require.NoError(t, err)
	assert.Equal(t, "你好", m.Body)
	assert.Equal(t, int64(100), pub.last.OrderID)
}

func TestSendMessage_EmptyBody(t *testing.T) {
	s, _, _ := newService()
	_, err := s.SendMessage(context.Background(), 100, 1, 2, "  ")
	assert.Error(t, err)
}

func TestSendMessage_BodyTooLong(t *testing.T) {
	s, _, _ := newService()
	long := make([]byte, 1001)
	for i := range long {
		long[i] = 'a'
	}
	_, err := s.SendMessage(context.Background(), 100, 1, 2, string(long))
	assert.Error(t, err)
}

func TestSendMessage_FromEqualsTo(t *testing.T) {
	s, _, _ := newService()
	_, err := s.SendMessage(context.Background(), 100, 1, 1, "x")
	assert.Error(t, err)
}

func TestSendMessage_MissingIDs(t *testing.T) {
	s, _, _ := newService()
	_, err := s.SendMessage(context.Background(), 0, 1, 2, "x")
	assert.Error(t, err)
	_, err = s.SendMessage(context.Background(), 100, 0, 2, "x")
	assert.Error(t, err)
	_, err = s.SendMessage(context.Background(), 100, 1, 0, "x")
	assert.Error(t, err)
}

func TestListByOrder(t *testing.T) {
	s, _, _ := newService()
	for i := 0; i < 3; i++ {
		_, err := s.SendMessage(context.Background(), 100, 1, 2, "msg")
		require.NoError(t, err)
	}
	list, err := s.ListByOrder(context.Background(), 100, 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestListByOrder_MissingID(t *testing.T) {
	s, _, _ := newService()
	_, err := s.ListByOrder(context.Background(), 0, 10, 0)
	assert.Error(t, err)
}

func TestSendMessage_NilPublisher(t *testing.T) {
	s := New(&fakeRepo{msgs: map[int64]*Message{}}, nil)
	_, err := s.SendMessage(context.Background(), 100, 1, 2, "x")
	assert.NoError(t, err)
}