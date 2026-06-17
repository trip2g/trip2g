package usertoken

import (
	"context"
	"testing"

	"github.com/valyala/fasthttp"
)

func newManager() *Manager {
	return NewManager(Config{
		CookieName: "trip2g_token",
		Secret:     "test-secret",
		ExpiresIn:  3600e9,
		Insecure:   true,
	})
}

// ParseToken verifies a raw session-token string (e.g. from Authorization:
// Bearer) without a cookie. A valid token returns its Data; an invalid/garbage
// token returns (nil, nil) so callers treat it as anonymous.
func TestParseToken(t *testing.T) {
	mgr := newManager()

	stored, err := mgr.Store(&fasthttp.RequestCtx{}, Data{ID: 42, Role: "admin"})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	got, err := mgr.ParseToken(context.Background(), stored.JWT)
	if err != nil {
		t.Fatalf("ParseToken(valid): unexpected err: %v", err)
	}
	if got == nil {
		t.Fatal("ParseToken(valid): got nil, want Data")
	}
	if got.ID != 42 || got.Role != "admin" || !got.IsAdmin() {
		t.Fatalf("ParseToken(valid): got %+v, want {42 admin}", got)
	}

	garbage, err := mgr.ParseToken(context.Background(), "not-a-jwt")
	if err != nil {
		t.Fatalf("ParseToken(garbage): unexpected err: %v", err)
	}
	if garbage != nil {
		t.Fatalf("ParseToken(garbage): got %+v, want nil (anonymous)", garbage)
	}
}
