package webhookutil

import (
	"net/url"
	"strings"
)

// RequireHTTPS returns a non-empty error message if the given URL uses a plain
// http:// scheme on a non-loopback host. It is silent for loopback addresses
// (localhost, 127.0.0.1, ::1) to ease local development.
//
// Call RequireHTTPSUnlessDevMode when the caller has access to a dev-mode flag
// (e.g. e2e Docker compose where the fleet container is reachable only via its
// compose service name, which is not a loopback address).
func RequireHTTPS(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") {
		return "" // https:// or non-http: let other validators handle it
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "" // malformed URL handled elsewhere
	}
	h := u.Hostname()
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return "" // loopback allowed in dev
	}
	return "https is required when pass_api_key is enabled (secrets and tokens must not travel over cleartext)"
}

// RequireHTTPSUnlessDevMode is like RequireHTTPS but skips the check when
// devMode is true. Use this in use-case resolvers that have access to the
// app's DevMode flag so that e2e Docker compose environments (where the
// fleet's callback URL uses a Docker compose service name, not a loopback)
// can register plain-HTTP webhooks without disabling security in production.
func RequireHTTPSUnlessDevMode(rawURL string, devMode bool) string {
	if devMode {
		return ""
	}
	return RequireHTTPS(rawURL)
}
