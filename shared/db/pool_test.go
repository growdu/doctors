package db_test

import (
	"context"
	"testing"

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
}