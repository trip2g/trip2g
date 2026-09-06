package appresp

import (
	"net/url"
	"strings"
)

// SamePath answers a Location that can only be this site: target reduced to its
// path, query and fragment, or "/" when it names anywhere else.
//
// The value is parsed and rebuilt rather than inspected and passed on, because
// a target that merely starts with a slash is not yet same-origin and the ways
// it can stop being one do not enumerate. "//evil.com" is protocol-relative. A
// browser strips ASCII tab, LF and CR from a URL before parsing it, so
// "/\t/evil.com" becomes "//evil.com" in the only place that matters, and CR LF
// in a header value is worse than a redirect. url.Parse refuses control
// characters as a class, EscapedPath re-encodes everything else that could be
// read a second way, and a host anywhere in the target is refused outright.
//
// One rule in one place: everything that builds a Location out of something a
// request carried goes through here.
func SamePath(target string) string {
	parsed, err := url.Parse(target)
	if err != nil || parsed.Scheme != "" || parsed.Opaque != "" || parsed.Host != "" {
		return "/"
	}

	path := parsed.EscapedPath()
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.HasPrefix(path, "/\\") {
		return "/"
	}

	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}

	if parsed.Fragment != "" {
		path += "#" + parsed.EscapedFragment()
	}

	return path
}
