package clientip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func requestFrom(remote string, headers map[string]string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{IP: net.ParseIP(remote), Port: 40000}, nil)

	return ctx
}

func TestProxiesOf(t *testing.T) {
	var proxies Proxies
	require.NoError(t, proxies.Set("127.0.0.1, 172.16.0.0/12, fd00::/8"))

	tests := []struct {
		name    string
		remote  string
		headers map[string]string
		want    string
	}{
		{
			name:   "no proxy, no headers",
			remote: "203.0.113.7",
			want:   "203.0.113.7",
		},
		{
			name:    "untrusted peer spoofing X-Real-IP",
			remote:  "203.0.113.7",
			headers: map[string]string{"X-Real-IP": "198.51.100.1"},
			want:    "203.0.113.7",
		},
		{
			name:    "untrusted peer spoofing X-Forwarded-For",
			remote:  "203.0.113.7",
			headers: map[string]string{"X-Forwarded-For": "198.51.100.1"},
			want:    "203.0.113.7",
		},
		{
			name:    "trusted proxy, X-Real-IP",
			remote:  "172.18.0.5",
			headers: map[string]string{"X-Real-IP": "198.51.100.1", "X-Forwarded-For": "192.0.2.9"},
			want:    "198.51.100.1",
		},
		{
			name:    "trusted proxy, X-Forwarded-For takes the right-most untrusted hop",
			remote:  "127.0.0.1",
			headers: map[string]string{"X-Forwarded-For": "6.6.6.6, 198.51.100.1, 172.18.0.5"},
			want:    "198.51.100.1",
		},
		{
			name:   "trusted proxy without headers",
			remote: "127.0.0.1",
			want:   "127.0.0.1",
		},
		{
			name:    "trusted proxy with a garbage header",
			remote:  "127.0.0.1",
			headers: map[string]string{"X-Real-IP": "not an ip", "X-Forwarded-For": "nope"},
			want:    "127.0.0.1",
		},
		{
			name:    "IPv6 trusted proxy",
			remote:  "fd12::1",
			headers: map[string]string{"X-Real-IP": "2001:db8::42"},
			want:    "2001:db8::42",
		},
		{
			name:    "IPv6 untrusted peer",
			remote:  "2001:db8::1",
			headers: map[string]string{"X-Real-IP": "2001:db8::42"},
			want:    "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, proxies.Of(requestFrom(tt.remote, tt.headers)))
		})
	}
}

func TestProxiesOf_NoneTrustedByDefault(t *testing.T) {
	var proxies Proxies

	got := proxies.Of(requestFrom("127.0.0.1", map[string]string{"X-Real-IP": "198.51.100.1"}))
	require.Equal(t, "127.0.0.1", got)
}

func TestProxiesSet(t *testing.T) {
	var proxies Proxies
	require.NoError(t, proxies.Set("10.1.2.3/8,::1"))
	require.Equal(t, "10.0.0.0/8,::1/128", proxies.String())

	for _, bad := range []string{"10.0.0.0/33", "localhost", "10.0.0.1,example.com", "1.2.3"} {
		require.Error(t, proxies.Set(bad), bad)
	}
}
