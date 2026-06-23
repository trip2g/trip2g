package readreplica

import (
	"net/http"
	"testing"
	"time"
)

func TestIsWrite(t *testing.T) {
	cases := map[string]bool{
		http.MethodGet:     false,
		http.MethodHead:    false,
		http.MethodOptions: false,
		http.MethodPost:    true,
		http.MethodPut:     true,
		http.MethodPatch:   true,
		http.MethodDelete:  true,
		"PROPFIND":         true, // unknown/non-safe methods forward to leader by default
	}

	for method, want := range cases {
		if got := IsWrite(method); got != want {
			t.Errorf("IsWrite(%q) = %v, want %v", method, got, want)
		}
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	secret := "test-secret"
	now := time.Unix(1_700_000_000, 0)

	token := SignAuth(secret, now, time.Minute)
	if token == "" {
		t.Fatal("SignAuth returned empty token")
	}

	if err := VerifyAuth(secret, token, now.Add(30*time.Second)); err != nil {
		t.Errorf("VerifyAuth within TTL failed: %v", err)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	secret := "test-secret"
	now := time.Unix(1_700_000_000, 0)
	token := SignAuth(secret, now, time.Minute)

	if err := VerifyAuth(secret, token, now.Add(2*time.Minute)); err == nil {
		t.Error("VerifyAuth accepted an expired token")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token := SignAuth("right-secret", now, time.Minute)

	if err := VerifyAuth("wrong-secret", token, now); err == nil {
		t.Error("VerifyAuth accepted a token signed with a different secret")
	}
}

func TestVerifyRejectsTampered(t *testing.T) {
	secret := "test-secret"
	now := time.Unix(1_700_000_000, 0)
	token := SignAuth(secret, now, time.Minute)

	tampered := token + "x"
	if err := VerifyAuth(secret, tampered, now); err == nil {
		t.Error("VerifyAuth accepted a tampered token")
	}

	if err := VerifyAuth(secret, "garbage", now); err == nil {
		t.Error("VerifyAuth accepted a malformed token")
	}
}
