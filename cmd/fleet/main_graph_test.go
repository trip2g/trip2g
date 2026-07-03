package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLoopbackAddr(t *testing.T) {
	tests := []struct {
		addr string
		ok   bool
	}{
		{"127.0.0.1:9092", true},
		{"localhost:9092", true},
		{"[::1]:9092", true},
		{"127.0.0.2:9092", true}, // whole 127/8 is loopback
		{":9092", false},         // empty host binds all interfaces
		{"0.0.0.0:9092", false},
		{"192.168.1.10:9092", false},
		{"[::]:9092", false},
		{"example.com:9092", false},
		{"127.0.0.1", false}, // missing port
	}
	for _, tt := range tests {
		err := validateLoopbackAddr(tt.addr)
		if tt.ok {
			require.NoError(t, err, tt.addr)
		} else {
			require.Error(t, err, tt.addr)
		}
	}
}
