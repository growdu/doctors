package kafka_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/kafka"
)

func TestValidateBrokers(t *testing.T) {
	cases := []struct {
		name    string
		brokers []string
		valid   bool
	}{
		{"one", []string{"localhost:9092"}, true},
		{"multi", []string{"a:9092", "b:9092"}, true},
		{"empty", []string{}, false},
		{"nil", nil, false},
		{"invalid", []string{"localhost"}, false},
		{"mixed", []string{"localhost:9092", "broken"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := kafka.ValidateBrokers(tc.brokers)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNormalizeTopic(t *testing.T) {
	cases := []struct{ in, want string }{
		{"order.created", "order.created"},
		{"order.created.v2", "order.created.v2"},
		{"ORDER.CREATED", "order.created"},
		{"  spaced  ", "spaced"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, kafka.NormalizeTopic(tc.in))
		})
	}
}

func TestNormalizeTopic_RejectsInvalid(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/foo", ""},
		{"foo/", ""},
		{"foo bar", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, kafka.NormalizeTopic(tc.in))
		})
	}
}