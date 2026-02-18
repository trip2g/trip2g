package model

import (
	"net/url"
	"strings"
)

// ParsedRoute is a single route from frontmatter.
type ParsedRoute struct {
	Host string // "" = main domain.
	Path string // "" = use note's Permalink at registration; "/" = explicit root; "/x" = explicit path.
}

// ParseRoute parses a route string from frontmatter.
//
// Examples:
//
//	"/about"        -> {Host: "",           Path: "/about"}  main domain alias.
//	"/"             -> {Host: "",           Path: "/"}       main domain root.
//	"foo.com"       -> {Host: "foo.com",    Path: ""}        custom domain, path = note's Permalink.
//	"foo.com/"      -> {Host: "foo.com",    Path: "/"}       custom domain root (explicit).
//	"foo.com/hello" -> {Host: "foo.com",    Path: "/hello"}  custom domain, explicit path.
func ParseRoute(value string) ParsedRoute {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/") {
		return ParsedRoute{Host: "", Path: value}
	}
	idx := strings.Index(value, "/")
	if idx == -1 {
		// "foo.com" - no explicit path, use note's Permalink at registration time.
		return ParsedRoute{Host: NormalizeDomain(value), Path: ""}
	}
	return ParsedRoute{Host: NormalizeDomain(value[:idx]), Path: value[idx:]}
}

// NormalizeDomain lowercases and strips www. prefix.
func NormalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	d = strings.TrimPrefix(d, "www.")
	return d
}

// ExtractHost parses a URL and returns the hostname (without port for standard ports).
func ExtractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}

// ExtractRoutes parses route/routes frontmatter fields into ParsedRoute slice.
func (n *NoteView) ExtractRoutes() []ParsedRoute {
	seen := make(map[string]struct{})
	var routes []ParsedRoute

	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		r := ParseRoute(value)
		key := r.Host + "|" + r.Path
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			routes = append(routes, r)
		}
	}

	if v, ok := n.RawMeta["route"].(string); ok {
		add(v)
	}

	switch v := n.RawMeta["routes"].(type) {
	case string:
		add(v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	}

	return routes
}
