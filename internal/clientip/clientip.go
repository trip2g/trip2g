// Package clientip finds the address of the client behind a request, trusting
// X-Real-IP and X-Forwarded-For only when the connection comes from a
// configured proxy.
package clientip

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/valyala/fasthttp"
)

// Proxies are the peers whose forwarding headers are believed. As a
// flag.Value it reads a comma-separated list of IPs and CIDRs.
type Proxies []netip.Prefix

func (p *Proxies) String() string {
	parts := make([]string, len(*p))
	for i, prefix := range *p {
		parts[i] = prefix.String()
	}
	return strings.Join(parts, ",")
}

func (p *Proxies) Set(value string) error {
	var parsed Proxies

	for _, entry := range strings.Split(value, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		prefix, err := parseEntry(entry)
		if err != nil {
			return err
		}

		parsed = append(parsed, prefix)
	}

	*p = parsed
	return nil
}

func parseEntry(entry string) (netip.Prefix, error) {
	if strings.Contains(entry, "/") {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("invalid CIDR %q: %w", entry, err)
		}
		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(entry)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid IP %q: %w", entry, err)
	}
	addr = addr.Unmap()

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

func (p Proxies) trusts(addr netip.Addr) bool {
	for _, prefix := range p {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// Of returns the client address of the request. From a trusted proxy it is
// X-Real-IP, else the right-most X-Forwarded-For hop that is not itself a
// trusted proxy; from anyone else it is the connection's address.
func (p Proxies) Of(ctx *fasthttp.RequestCtx) string {
	remote, ok := netip.AddrFromSlice(ctx.RemoteIP())
	if !ok {
		return ""
	}
	remote = remote.Unmap()

	if !p.trusts(remote) {
		return remote.String()
	}

	realIP, err := netip.ParseAddr(strings.TrimSpace(string(ctx.Request.Header.Peek("X-Real-IP"))))
	if err == nil {
		return realIP.Unmap().String()
	}

	hops := strings.Split(string(ctx.Request.Header.Peek("X-Forwarded-For")), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, hopErr := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if hopErr != nil {
			break
		}

		hop = hop.Unmap()
		if !p.trusts(hop) {
			return hop.String()
		}
	}

	return remote.String()
}
