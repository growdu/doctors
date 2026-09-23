//go:build integration
// +build integration

package redis_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/shared/redis"
)

func redisAddr() string {
	if v := os.Getenv("DOCTORS_REDIS_ADDR"); v != "" {
		return v
	}
	return "localhost:6379"
}

func TestIntegration_RedisPingAndSetGet(t *testing.T) {
	ctx := context.Background()
	c, err := redis.NewClient(redis.Config{Addr: redisAddr()})
	require.NoError(t, err)
	defer c.Close()

	require.NoError(t, redis.HealthCheck(ctx, c))

	require.NoError(t, c.Set(ctx, "doctors:test", "ok", 0).Err())
	v, err := c.Get(ctx, "doctors:test").Result()
	require.NoError(t, err)
	require.Equal(t, "ok", v)
	c.Del(ctx, "doctors:test")
}