package redis_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/redis"
)

func TestValidateAddr(t *testing.T) {
	cases := []struct {
		addr  string
		valid bool
	}{
		{"localhost:6379", true},
		{"127.0.0.1:6379", true},
		{"redis://localhost:6379", false}, // go-redis v9 不带 scheme
		{"", false},
		{":", false},
	}
	for _, tc := range cases {
		t.Run(tc.addr, func(t *testing.T) {
			err := redis.ValidateAddr(tc.addr)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := redis.Config{Addr: "localhost:6379"}
	cfg.ApplyDefaults()
	assert.Equal(t, 0, cfg.DB)
}