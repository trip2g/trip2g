package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type timeoutError struct{}

func (timeoutError) Error() string { return "i/o timeout" }
func (timeoutError) Timeout() bool { return true }

func TestFederatedStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil is ok", nil, "ok"},
		{"deadline exceeded is timeout", context.DeadlineExceeded, "timeout"},
		{"wrapped deadline is timeout", errors.New("call: " + context.DeadlineExceeded.Error()), "error"},
		{"net timeout is timeout", timeoutError{}, "timeout"},
		{"other error is error", errors.New("boom"), "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, federatedStatus(tc.err))
		})
	}
}

func TestErrorReason(t *testing.T) {
	cases := []struct {
		name string
		code int
		want string
	}{
		{"invalid params", ErrCodeInvalidParams, "invalid_params"},
		{"method not found", ErrCodeMethodNotFound, "not_found"},
		{"internal", ErrCodeInternal, "internal"},
		{"unknown code", -1, "other"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, errorReason(tc.code))
		})
	}
}
