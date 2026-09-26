package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/db"
)

func TestValidateDSN(t *testing.T) {
	cases := []struct {
		dsn   string
		valid bool
	}{
		{"postgres://u:p@localhost:5432/d", true},
		{"postgresql://u:p@localhost:5432/d", true},
		{"", false},
		{"mysql://...", false},
		{"not-a-dsn", false},
	}
	for _, tc := range cases {
		t.Run(tc.dsn, func(t *testing.T) {
			err := db.ValidateDSN(tc.dsn)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNewPool_InvalidDSNReturnsError(t *testing.T) {
	pool, err := db.NewPool(context.Background(), db.Config{DSN: "invalid"})
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestNewPool_EmptyDSNReturnsError(t *testing.T) {
	pool, err := db.NewPool(context.Background(), db.Config{DSN: ""})
	assert.Error(t, err)
	assert.Nil(t, pool)
}

func TestConfig_Defaults(t *testing.T) {
	cfg := db.Config{DSN: "postgres://x"}
	cfg.ApplyDefaults()
	assert.Equal(t, int32(10), cfg.MaxConns)
	assert.Equal(t, int32(2), cfg.MinConns)
	// §31 增量：补齐 ConnectTimeout / HealthCheckPeriod / 生命周期默认值。
	assert.Equal(t, 5*time.Second, cfg.ConnectTimeout)
	assert.Equal(t, 30*time.Second, cfg.HealthCheckPeriod)
	assert.Equal(t, time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 10*time.Minute, cfg.MaxConnIdleTime)
}

func TestConfig_CustomTimeouts(t *testing.T) {
	// 调用方显式传入自定义值 → ApplyDefaults 不覆盖。
	cfg := db.Config{
		DSN:               "postgres://x",
		MaxConns:          4,
		MinConns:          1,
		ConnectTimeout:    2 * time.Second,
		HealthCheckPeriod: time.Minute,
		MaxConnLifetime:   2 * time.Hour,
		MaxConnIdleTime:   5 * time.Minute,
	}
	cfg.ApplyDefaults()
	assert.Equal(t, int32(4), cfg.MaxConns)
	assert.Equal(t, int32(1), cfg.MinConns)
	assert.Equal(t, 2*time.Second, cfg.ConnectTimeout)
	assert.Equal(t, time.Minute, cfg.HealthCheckPeriod)
	assert.Equal(t, 2*time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 5*time.Minute, cfg.MaxConnIdleTime)
}