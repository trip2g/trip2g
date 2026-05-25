package ssrfsafe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"
)

// ErrBlockedAddress is returned when a resolved IP falls in a private/internal range.
var ErrBlockedAddress = errors.New("ssrfsafe: address is blocked (private/internal)")

// blockedPrefixes covers RFC1918, loopback, link-local, cloud metadata, and
// IPv6 ranges that can encode private IPv4 (mapped, NAT64, 6to4).
var blockedPrefixes = []netip.Prefix{
	// IPv4
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),    // carrier-grade NAT (RFC6598)
	netip.MustParsePrefix("127.0.0.0/8"),      // loopback
	netip.MustParsePrefix("169.254.0.0/16"),   // link-local / AWS+GCP metadata
	netip.MustParsePrefix("172.16.0.0/12"),    // RFC1918
	netip.MustParsePrefix("192.0.0.0/24"),     // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),     // TEST-NET-1
	netip.MustParsePrefix("192.168.0.0/16"),   // RFC1918
	netip.MustParsePrefix("198.18.0.0/15"),    // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"),  // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),   // TEST-NET-3
	netip.MustParsePrefix("224.0.0.0/4"),      // multicast
	netip.MustParsePrefix("240.0.0.0/4"),      // reserved
	netip.MustParsePrefix("255.255.255.255/32"),
	// IPv6
	netip.MustParsePrefix("::1/128"),          // loopback
	netip.MustParsePrefix("fc00::/7"),         // unique local (RFC4193)
	netip.MustParsePrefix("fe80::/10"),        // link-local
	netip.MustParsePrefix("::ffff:0:0/96"),    // IPv4-mapped (can encode private IPv4)
	netip.MustParsePrefix("64:ff9b::/96"),     // NAT64 (can map to private IPv4)
	netip.MustParsePrefix("2001:db8::/32"),    // documentation
	netip.MustParsePrefix("2002::/16"),        // 6to4 (can encode private IPv4)
}

func isSafe(ip netip.Addr) bool {
	ip = ip.Unmap() // normalise ::ffff:x.x.x.x → plain IPv4
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// DialTimeout is a fasthttp-compatible DialFuncWithTimeout that resolves the
// target host, rejects any private/internal IP (including all resolved
// addresses — not just the first), then dials the first resolved IP directly
// to prevent DNS rebinding between check and connect.
func DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("ssrfsafe: parse addr: %w", err)
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("ssrfsafe: resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("ssrfsafe: no addresses for %s", host)
	}

	// Reject the entire request if ANY resolved IP is private.
	// An attacker could return both a safe and a private IP.
	for _, ipAddr := range ips {
		ip, ok := netip.AddrFromSlice(ipAddr.IP)
		if !ok {
			return nil, fmt.Errorf("ssrfsafe: invalid IP from resolver: %v", ipAddr.IP)
		}
		if !isSafe(ip) {
			return nil, fmt.Errorf("%w: %s resolves to %s", ErrBlockedAddress, host, ip.Unmap())
		}
	}

	// Dial the first resolved IP directly — bypasses DNS re-resolution on connect.
	dialAddr := net.JoinHostPort(ips[0].IP.String(), port)
	return (&net.Dialer{}).DialContext(ctx, "tcp", dialAddr)
}
