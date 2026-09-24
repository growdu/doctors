package events

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/repo"
)

// TestNopPublisher_CountsCreated 验证 PublishOrderCreated 计数。
func TestNopPublisher_CountsCreated(t *testing.T) {
	p := &NopPublisher{}
	require.NoError(t, p.PublishOrderCreated(context.Background(), &repo.Order{ID: 1}))
	require.NoError(t, p.PublishOrderCreated(context.Background(), &repo.Order{ID: 2}))
	assert.Equal(t, 2, p.CreatedCount)
}

// TestNopPublisher_CountsAccepted 验证 PublishOrderAccepted 计数。
func TestNopPublisher_CountsAccepted(t *testing.T) {
	p := &NopPublisher{}
	require.NoError(t, p.PublishOrderAccepted(context.Background(), &repo.Order{ID: 1}))
	assert.Equal(t, 1, p.AcceptedCount)
}

// TestNopPublisher_Close 不报错。
func TestNopPublisher_Close(t *testing.T) {
	p := &NopPublisher{}
	assert.NoError(t, p.Close())
}

// TestKafkaPublisher_NilWriterReturnsError 直接用 nil writer 不应 panic，应返回错误。
func TestKafkaPublisher_NilWriterReturnsError(t *testing.T) {
	var p *KafkaPublisher
	err := p.PublishOrderCreated(context.Background(), &repo.Order{ID: 1})
	assert.Error(t, err)
}