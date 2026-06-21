package handleoidccallback

import (
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/oidcauth"
)

type accountOutcome struct {
	Provision bool
	Reject    bool
	BError    string
}

// decideAccount decides what to do when the OIDC email has NO existing trip2g user.
func decideAccount(creds db.OidcCredential, info *oidcauth.UserInfo) accountOutcome {
	if !creds.AutoProvision {
		return accountOutcome{Reject: true, BError: "user_not_found"}
	}

	if !info.EmailVerified {
		return accountOutcome{Reject: true, BError: "email_not_allowed"}
	}

	if creds.AllowedEmailDomain != "" {
		domain := ""
		if idx := strings.LastIndex(info.Email, "@"); idx >= 0 {
			domain = info.Email[idx+1:]
		}
		if strings.ToLower(domain) != strings.ToLower(creds.AllowedEmailDomain) {
			return accountOutcome{Reject: true, BError: "email_not_allowed"}
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
			return accountOutcome{Reject: true, BError: "email_not_allowed"}
		}
	}

	return accountOutcome{Provision: true}
}
