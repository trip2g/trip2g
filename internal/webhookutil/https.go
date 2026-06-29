package webhookutil

import (
	"net/url"
	"strings"
)

// RequireHTTPS returns a non-empty error message if the given URL uses a plain
// http:// scheme on a non-loopback host. It is silent for loopback addresses
// (localhost, 127.0.0.1, ::1) to ease local development.
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
