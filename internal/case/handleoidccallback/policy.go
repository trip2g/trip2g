package handleoidccallback

import (
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/oidcauth"
)

// berror codes returned to the sign-in page via ?berror=.
const (
	berrUserNotFound    = "user_not_found"
	berrEmailNotAllowed = "email_not_allowed"
)

// accessBError returns a non-empty berror code if a configured access gate
// rejects this identity. Applies to EVERY login (existing users and new).
// Empty allowed_email_domain / required_group are no-ops.
func accessBError(creds db.OidcCredential, info *oidcauth.UserInfo) string {
	if creds.AllowedEmailDomain != "" {
		// domain = case-insensitive part after the last '@'
		at := strings.LastIndex(info.Email, "@")
		domain := ""
		if at >= 0 {
			domain = strings.ToLower(info.Email[at+1:])
		}
		if domain != strings.ToLower(creds.AllowedEmailDomain) {
			return berrEmailNotAllowed
		}
	}
	if creds.RequiredGroup != "" {
		found := false
		for _, g := range info.Groups {
			if g == creds.RequiredGroup {
				found = true
				break
			}
		}
		if !found {
			return berrEmailNotAllowed
		}
	}
	return ""
}

// provisionBError returns a non-empty berror code if a NEW (not-yet-existing)
// OIDC identity may not be auto-provisioned into an account.
func provisionBError(creds db.OidcCredential, info *oidcauth.UserInfo) string {
	if !creds.AutoProvision {
		return berrUserNotFound
	}
	if !info.EmailVerified {
		return berrEmailNotAllowed // verify email before creating an account
	}
	return ""
}
