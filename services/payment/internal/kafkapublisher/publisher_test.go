package kafkapublisher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/contracts"
)

// TestPublisher_NilWriterReturnsError 验证 nil publisher 的 publish 路径返回 error。
func TestPublisher_NilWriterReturnsError(t *testing.T) {
	var p *Publisher
	err := p.PublishPaymentCompleted(context.Background(), contracts.PaymentCompletedEvent{PaymentID: 1})
	assert.Error(t, err)
	err = p.PublishPaymentRefunded(context.Background(), contracts.PaymentRefundedEvent{RefundID: 1})
	assert.Error(t, err)
}

// TestPublisher_NilCloseSafe 验证 nil publisher Close 不 panic。
func TestPublisher_NilCloseSafe(t *testing.T) {
	var p *Publisher
	assert.NoError(t, p.Close())
}

// TestNew_InvalidBrokersReturnsError 验证 brokers 非法时 New 返回 error。
func TestNew_InvalidBrokersReturnsError(t *testing.T) {
	p, err := New(nil)
	assert.Error(t, err)
	assert.Nil(t, p)
}