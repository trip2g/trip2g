package handleoidccallback

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/oidcauth"
)

func TestAccessBError(t *testing.T) {
	tests := []struct {
		name  string
		creds db.OidcCredential
		info  *oidcauth.UserInfo
		want  string
	}{
		{
			name:  "no gates configured allows",
			creds: db.OidcCredential{},
			info:  verifiedUserInfo("a@example.com"),
			want:  "",
		},
		{
			name:  "allowed domain match (exact) allows",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  verifiedUserInfo("a@example.com"),
			want:  "",
		},
		{
			name:  "allowed domain mismatch rejects",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  verifiedUserInfo("a@other.com"),
			want:  "email_not_allowed",
		},
		{
			name:  "allowed domain match is case-insensitive (creds upper)",
			creds: db.OidcCredential{AllowedEmailDomain: "Example.COM"},
			info:  verifiedUserInfo("a@example.com"),
			want:  "",
		},
		{
			name:  "allowed domain match is case-insensitive (email upper)",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  verifiedUserInfo("a@EXAMPLE.com"),
			want:  "",
		},
		{
			name:  "uses domain after last @ (multiple @ in local part)",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  verifiedUserInfo("weird@local@example.com"),
			want:  "",
		},
		{
			name:  "no @ in email rejects when domain required",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  verifiedUserInfo("noatsign"),
			want:  "email_not_allowed",
		},
		{
			name:  "required group present allows",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  verifiedUserInfo("a@example.com", "editors", "admins"),
			want:  "",
		},
		{
			name:  "required group absent rejects",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  verifiedUserInfo("a@example.com", "editors"),
			want:  "email_not_allowed",
		},
		{
			name:  "required group with empty groups rejects",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  verifiedUserInfo("a@example.com"),
			want:  "email_not_allowed",
		},
		{
			name:  "domain and group both satisfied allows",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  verifiedUserInfo("a@example.com", "admins"),
			want:  "",
		},
		{
			name:  "domain ok but group missing rejects",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  verifiedUserInfo("a@example.com", "editors"),
			want:  "email_not_allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := accessBError(tt.creds, tt.info)
			require.Equal(t, tt.want, got)
		})
	}
}

type userByEmailLookupFunc func(context.Context, string) (db.User, error)

func (f userByEmailLookupFunc) UserByEmail(ctx context.Context, email string) (db.User, error) {
	return f(ctx, email)
}

func TestLookupAuthorizedUserRejectsBeforeAccountLookup(t *testing.T) {
	called := false
	lookup := userByEmailLookupFunc(func(context.Context, string) (db.User, error) {
		called = true
		return db.User{}, nil
	})

	_, err, berr := lookupAuthorizedUser(context.Background(), lookup, db.OidcCredential{}, &oidcauth.UserInfo{
		Email:         "existing@example.com",
		EmailVerified: false,
	})
	require.NoError(t, err)
	require.Equal(t, berrEmailNotAllowed, berr)
	require.False(t, called, "an unverified identity must not reach UserByEmail")
}

func TestAccessBErrorRejectsUnverifiedEmailForExistingAccountBinding(t *testing.T) {
	got := accessBError(db.OidcCredential{}, &oidcauth.UserInfo{
		Email:         "existing@example.com",
		EmailVerified: false,
	})
	require.Equal(t, berrEmailNotAllowed, got)
}

func TestSubjectBound(t *testing.T) {
	tests := []struct {
		name       string
		idTokenSub string
		userInfo   string
		want       bool
	}{
		{name: "matching subjects bind", idTokenSub: "user-123", userInfo: "user-123", want: true},
		{name: "mismatched subjects rejected", idTokenSub: "user-123", userInfo: "attacker-456", want: false},
		{name: "empty id_token subject rejected", idTokenSub: "", userInfo: "user-123", want: false},
		{name: "empty userinfo subject rejected", idTokenSub: "user-123", userInfo: "", want: false},
		{name: "both empty rejected", idTokenSub: "", userInfo: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, subjectBound(tt.idTokenSub, tt.userInfo))
		})
	}
}

func TestProvisionBError(t *testing.T) {
	tests := []struct {
		name  string
		creds db.OidcCredential
		info  *oidcauth.UserInfo
		want  string
	}{
		{
			name:  "auto_provision off rejects with user_not_found",
			creds: db.OidcCredential{AutoProvision: false},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  "user_not_found",
		},
		{
			name:  "auto_provision on but email not verified rejects",
			creds: db.OidcCredential{AutoProvision: true},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: false},
			want:  "email_not_allowed",
		},
		{
			name:  "auto_provision on and email verified allows",
			creds: db.OidcCredential{AutoProvision: true},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provisionBError(tt.creds, tt.info)
			require.Equal(t, tt.want, got)
		})
	}
}

func verifiedUserInfo(email string, groups ...string) *oidcauth.UserInfo {
	return &oidcauth.UserInfo{Email: email, EmailVerified: true, Groups: groups}
}
