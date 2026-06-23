package handleoidccallback

import (
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
			info:  &oidcauth.UserInfo{Email: "a@example.com"},
			want:  "",
		},
		{
			name:  "allowed domain match (exact) allows",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@example.com"},
			want:  "",
		},
		{
			name:  "allowed domain mismatch rejects",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@other.com"},
			want:  "email_not_allowed",
		},
		{
			name:  "allowed domain match is case-insensitive (creds upper)",
			creds: db.OidcCredential{AllowedEmailDomain: "Example.COM"},
			info:  &oidcauth.UserInfo{Email: "a@example.com"},
			want:  "",
		},
		{
			name:  "allowed domain match is case-insensitive (email upper)",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@EXAMPLE.com"},
			want:  "",
		},
		{
			name:  "uses domain after last @ (multiple @ in local part)",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "weird@local@example.com"},
			want:  "",
		},
		{
			name:  "no @ in email rejects when domain required",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "noatsign"},
			want:  "email_not_allowed",
		},
		{
			name:  "required group present allows",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", Groups: []string{"editors", "admins"}},
			want:  "",
		},
		{
			name:  "required group absent rejects",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", Groups: []string{"editors"}},
			want:  "email_not_allowed",
		},
		{
			name:  "required group with empty groups rejects",
			creds: db.OidcCredential{RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", Groups: nil},
			want:  "email_not_allowed",
		},
		{
			name:  "domain and group both satisfied allows",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", Groups: []string{"admins"}},
			want:  "",
		},
		{
			name:  "domain ok but group missing rejects",
			creds: db.OidcCredential{AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", Groups: []string{"editors"}},
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
