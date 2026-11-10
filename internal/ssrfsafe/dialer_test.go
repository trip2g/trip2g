package ssrfsafe

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSafe(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "8.8.8.8", want: true},
		{addr: "2606:4700:4700::1111", want: true},
		{addr: "::ffff:8.8.8.8", want: true},
		{addr: "127.0.0.1", want: false},
		{addr: "10.1.2.3", want: false},
		{addr: "169.254.169.254", want: false},
		{addr: "0.0.0.0", want: false},
		{addr: "::ffff:127.0.0.1", want: false},
		{addr: "::1", want: false},
		{addr: "::", want: false},
		{addr: "fd00::1", want: false},
		{addr: "fe80::1", want: false},
		{addr: "64:ff9b::a00:1", want: false},
		{addr: "2002:a00:1::", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			require.Equal(t, tt.want, isSafe(netip.MustParseAddr(tt.addr)))
		})
	}
}
