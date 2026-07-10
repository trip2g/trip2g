package handleoidccallback

import (
	"context"
	"strings"

	"trip2g/internal/db"
	"trip2g/internal/oidcauth"
)

type userByEmailLookup interface {
	UserByEmail(ctx context.Context, email string) (db.User, error)
}

// berror codes returned to the sign-in page via ?berror=.
const (
	berrUserNotFound     = "user_not_found"
	berrEmailNotAllowed  = "email_not_allowed"
	berrEmailNotVerified = "email_not_verified"
)

const (
	emailVerificationSourceUserInfo = "userinfo"
	emailVerificationSourceIDToken  = "id_token"

	emailVerificationReasonUserInfoEmailEmpty = "userinfo_email_missing"
	emailVerificationReasonUserInfoInvalid    = "userinfo_claim_invalid"
	emailVerificationReasonUserInfoFalse      = "userinfo_claim_false"
	emailVerificationReasonIDTokenMissing     = "id_token_claim_missing" //nolint:gosec // G101 false positive: OIDC reason code, not a credential
	emailVerificationReasonIDTokenInvalid     = "id_token_claim_invalid"
	emailVerificationReasonIDTokenFalse       = "id_token_claim_false"   //nolint:gosec // G101 false positive: OIDC reason code, not a credential
	emailVerificationReasonIDTokenEmailEmpty  = "id_token_email_missing" //nolint:gosec // G101 false positive: OIDC reason code, not a credential
	emailVerificationReasonEmailMismatch      = "id_token_userinfo_email_mismatch"
)

type emailVerificationDecision struct {
	Verified bool
	Source   string
	Reason   string
}

// resolveEmailVerification applies the OIDC email-verification precedence.
// UserInfo is authoritative when it contains email_verified. The signature-
// verified ID token is a fallback only when UserInfo omits the claim, and only
// when the ID-token email exactly matches the UserInfo email.
func resolveEmailVerification(
	idClaims *oidcauth.IDTokenClaims,
	info *oidcauth.UserInfo,
) emailVerificationDecision {
	if info == nil || info.Email == "" {
		return emailVerificationDecision{Reason: emailVerificationReasonUserInfoEmailEmpty}
	}

	if info.EmailVerified.Present {
		if !info.EmailVerified.Valid {
			return emailVerificationDecision{Reason: emailVerificationReasonUserInfoInvalid}
		}
		if info.EmailVerified.Value {
			return emailVerificationDecision{Verified: true, Source: emailVerificationSourceUserInfo}
		}
		return emailVerificationDecision{Reason: emailVerificationReasonUserInfoFalse}
	}

	if idClaims == nil || !idClaims.EmailVerified.Present {
		return emailVerificationDecision{Reason: emailVerificationReasonIDTokenMissing}
	}
	if !idClaims.EmailVerified.Valid {
		return emailVerificationDecision{Reason: emailVerificationReasonIDTokenInvalid}
	}
	if !idClaims.EmailVerified.Value {
		return emailVerificationDecision{Reason: emailVerificationReasonIDTokenFalse}
	}
	if idClaims.Email == "" {
		return emailVerificationDecision{Reason: emailVerificationReasonIDTokenEmailEmpty}
	}
	if idClaims.Email != info.Email {
		return emailVerificationDecision{Reason: emailVerificationReasonEmailMismatch}
	}
	return emailVerificationDecision{Verified: true, Source: emailVerificationSourceIDToken}
}

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

// lookupAuthorizedUser keeps the verified-email/access decision in front of
// the email-based account lookup. That ordering is the security boundary: an
// unverified provider claim must never select an existing local account.
func lookupAuthorizedUser(
	ctx context.Context,
	env userByEmailLookup,
	creds db.OidcCredential,
	info *oidcauth.UserInfo,
	verification emailVerificationDecision,
) (db.User, string, error) {
	if !verification.Verified {
		return db.User{}, berrEmailNotVerified, nil
	}
	if berr := accessBError(creds, info); berr != "" {
		return db.User{}, berr, nil
	}
	user, err := env.UserByEmail(ctx, info.Email)
	return user, "", err
}

// subjectBound reports whether the id_token subject and the userinfo subject
// identify the same principal. Per OIDC Core 5.3.2 the id_token sub MUST match
// the userinfo sub; empty subjects never bind.
func subjectBound(idTokenSub, userInfoSub string) bool {
	return idTokenSub != "" && userInfoSub != "" && idTokenSub == userInfoSub
}

// provisionBError returns a non-empty berror code if a NEW (not-yet-existing)
// OIDC identity may not be auto-provisioned into an account.
func provisionBError(creds db.OidcCredential, verification emailVerificationDecision) string {
	if !verification.Verified {
		return berrEmailNotVerified
	}
	if !creds.AutoProvision {
		return berrUserNotFound
	}
	return ""
}
