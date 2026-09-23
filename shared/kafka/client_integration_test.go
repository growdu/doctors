//go:build integration
// +build integration

package kafka_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/kafka"
)

func brokers() []string {
	if v := os.Getenv("DOCTORS_KAFKA_BROKERS"); v != "" {
		return []string{v}
	}
	return []string{"localhost:29092"}
}

// TestIntegration_ProduceConsume 真实跑一次 produce + consume 闭环。
func TestIntegration_ProduceConsume(t *testing.T) {
	topic := fmt.Sprintf("doctors.test.%d", time.Now().UnixNano())
	group := fmt.Sprintf("doctors-test-%d", time.Now().UnixNano())

	w, err := kafka.NewWriter(brokers(), topic)
	require.NoError(t, err)
	defer w.Close()

	r, err := kafka.NewReader(brokers(), topic, group)
	require.NoError(t, err)
	defer r.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// produce
	require.NoError(t, w.WriteMessages(ctx, kafka.Message{
		Key:   []byte("k1"),
		Value: []byte(`{"hello":"world"}`),
	}))

	// consume
	r.SetDeadline(time.Now().Add(8 * time.Second))
	msg, err := r.ReadMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, "k1", string(msg.Key))
	require.Equal(t, `{"hello":"world"}`, string(msg.Value))
}