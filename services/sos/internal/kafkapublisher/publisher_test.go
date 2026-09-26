package kafkapublisher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/contracts"
)

// TestPublisher_NilWriterReturnsError 验证 nil publisher 的 PublishSOSRaised 返回 error。
func TestPublisher_NilWriterReturnsError(t *testing.T) {
	var p *Publisher
	err := p.PublishSOSRaised(context.Background(), contracts.SOSRaisedEvent{SOSID: 1})
	assert.Error(t, err)
}

// TestPublisher_NilCloseSafe 验证 nil publisher Close 不 panic。
func TestPublisher_NilCloseSafe(t *testing.T) {
	var p *Publisher
	assert.NoError(t, p.Close())
}

// TestNew_InvalidBrokersReturnsError 验证 brokers 非法时 New 返回 error。
func TestNew_InvalidBrokersReturnsError(t *testing.T) {
	p, err := New(nil, contracts.TopicSOSRaised)
	assert.Error(t, err)
	assert.Nil(t, p)
}