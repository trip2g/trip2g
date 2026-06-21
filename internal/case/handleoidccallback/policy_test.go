package handleoidccallback

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/db"
	"trip2g/internal/oidcauth"
)

func TestDecideAccount(t *testing.T) {
	tests := []struct {
		name  string
		creds db.OidcCredential
		info  *oidcauth.UserInfo
		want  accountOutcome
	}{
		{
			name:  "auto_provision off rejects with user_not_found",
			creds: db.OidcCredential{AutoProvision: false},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  accountOutcome{Reject: true, BError: "user_not_found"},
		},
		{
			name:  "email not verified rejects with email_not_allowed",
			creds: db.OidcCredential{AutoProvision: true},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: false},
			want:  accountOutcome{Reject: true, BError: "email_not_allowed"},
		},
		{
			name:  "allowed domain mismatch rejects",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@other.com", EmailVerified: true},
			want:  accountOutcome{Reject: true, BError: "email_not_allowed"},
		},
		{
			name:  "allowed domain match (exact) provisions",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "allowed domain match is case-insensitive (creds upper)",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "Example.COM"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "allowed domain match is case-insensitive (email upper)",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "a@EXAMPLE.com", EmailVerified: true},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "uses domain after last @ (multiple @ in local part)",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com"},
			info:  &oidcauth.UserInfo{Email: "weird@local@example.com", EmailVerified: true},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "required group present provisions",
			creds: db.OidcCredential{AutoProvision: true, RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true, Groups: []string{"editors", "admins"}},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "required group absent rejects",
			creds: db.OidcCredential{AutoProvision: true, RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true, Groups: []string{"editors"}},
			want:  accountOutcome{Reject: true, BError: "email_not_allowed"},
		},
		{
			name:  "required group with empty groups rejects",
			creds: db.OidcCredential{AutoProvision: true, RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true, Groups: nil},
			want:  accountOutcome{Reject: true, BError: "email_not_allowed"},
		},
		{
			name:  "domain and group both satisfied provisions",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true, Groups: []string{"admins"}},
			want:  accountOutcome{Provision: true},
		},
		{
			name:  "domain ok but group missing rejects",
			creds: db.OidcCredential{AutoProvision: true, AllowedEmailDomain: "example.com", RequiredGroup: "admins"},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true, Groups: []string{"editors"}},
			want:  accountOutcome{Reject: true, BError: "email_not_allowed"},
		},
		{
			name:  "no gating, verified, auto-provision provisions",
			creds: db.OidcCredential{AutoProvision: true},
			info:  &oidcauth.UserInfo{Email: "a@example.com", EmailVerified: true},
			want:  accountOutcome{Provision: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideAccount(tt.creds, tt.info)
			require.Equal(t, tt.want, got)
		})
	}
}
