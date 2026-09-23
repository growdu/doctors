// Package kafka 提供 Kafka 生产者/消费者封装。
//
// 设计要点：
//   - 暴露 Writer / Reader，业务可直接使用 segmentio/kafka-go。
//   - Topic 名规范化：小写、去空格、不能为空或含非法字符。
//   - 集成测试用 //go:build integration 隔离。
package kafka

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

var topicPattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

// ValidateBrokers 校验 brokers 列表。
func ValidateBrokers(brokers []string) error {
	if len(brokers) == 0 {
		return errors.New("kafka: empty brokers")
	}
	for _, b := range brokers {
		if !strings.Contains(b, ":") {
			return fmt.Errorf("kafka: broker %q must be host:port", b)
		}
	}
	return nil
}

// NormalizeTopic 把 topic 规范化；非法返回空串。
func NormalizeTopic(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "" {
		return ""
	}
	if !topicPattern.MatchString(t) {
		return ""
	}
	return t
}

// NewWriter 构造生产者。
func NewWriter(brokers []string, topic string) (*kafka.Writer, error) {
	if err := ValidateBrokers(brokers); err != nil {
		return nil, err
	}
	if NormalizeTopic(topic) == "" {
		return nil, fmt.Errorf("kafka: invalid topic %q", topic)
	}
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 50 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}, nil
}

// NewReader 构造消费者。
func NewReader(brokers []string, topic, groupID string) (*kafka.Reader, error) {
	if err := ValidateBrokers(brokers); err != nil {
		return nil, err
	}
	if NormalizeTopic(topic) == "" {
		return nil, fmt.Errorf("kafka: invalid topic %q", topic)
	}
	if groupID == "" {
		return nil, errors.New("kafka: groupID required")
	}
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	}), nil
}