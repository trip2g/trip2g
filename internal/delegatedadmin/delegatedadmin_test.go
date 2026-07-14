package delegatedadmin_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trip2g/internal/delegatedadmin"
)

// newNext returns a handler that records whether it was called and always
// writes 200 OK.
func newNext(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("served"))
	})
}

// monolith spins up a fake /_system/graphql that captures the forwarded Cookie
// header and replies with the given role (or a custom raw handler).
func monolith(t *testing.T, gotCookie *string, respond func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_system/graphql" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if gotCookie != nil {
			*gotCookie = r.Header.Get("Cookie")
		}
		// Drain body so the caller's request is complete.
		_, _ = io.Copy(io.Discard, r.Body)
		respond(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func roleResponder(role string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"role":"` + role + `"}}}`))
	}
}

func doRequest(t *testing.T, mw *delegatedadmin.Middleware, cookie string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	var called bool
	h := mw.Wrap(newNext(&called))
	req := httptest.NewRequest(http.MethodGet, "/_fleet/graphql", nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr, called
}

func newMW(t *testing.T, baseURL string, client *http.Client) *delegatedadmin.Middleware {
	t.Helper()
	mw, err := delegatedadmin.New(delegatedadmin.Config{MonolithBaseURL: baseURL, HTTPClient: client})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return mw
}

func TestAdminCookieAllows(t *testing.T) {
	var gotCookie string
	srv := monolith(t, &gotCookie, roleResponder("ADMIN"))
	mw := newMW(t, srv.URL, nil)

	rr, called := doRequest(t, mw, "trip2g_token=abc.def.ghi")

	if !called {
		t.Fatal("expected next handler to be called for admin")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if gotCookie != "trip2g_token=abc.def.ghi" {
		t.Fatalf("cookie not forwarded verbatim: got %q", gotCookie)
	}
}

func TestNonAdminRejected(t *testing.T) {
	for _, role := range []string{"USER", "GUEST"} {
		srv := monolith(t, nil, roleResponder(role))
		mw := newMW(t, srv.URL, nil)

		rr, called := doRequest(t, mw, "trip2g_token=abc")

		if called {
			t.Fatalf("role %s: next must not be called", role)
		}
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("role %s: expected 401, got %d", role, rr.Code)
		}
	}
}

func TestNoCookieRejectedWithoutRoundTrip(t *testing.T) {
	var reached bool
	srv := monolith(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.Write([]byte(`{"data":{"viewer":{"role":"ADMIN"}}}`))
	})
	mw := newMW(t, srv.URL, nil)

	rr, called := doRequest(t, mw, "")

	if called {
		t.Fatal("next must not be called without a cookie")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Fatal("monolith should not be contacted when session cookie is absent")
	}
}

func TestUnrelatedCookieButNoSessionCookieRejected(t *testing.T) {
	var reached bool
	srv := monolith(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.Write([]byte(`{"data":{"viewer":{"role":"ADMIN"}}}`))
	})
	mw := newMW(t, srv.URL, nil)

	rr, called := doRequest(t, mw, "other=1; unrelated=2")

	if called {
		t.Fatal("next must not be called without the session cookie")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if reached {
		t.Fatal("monolith should not be contacted without the session cookie")
	}
}

func TestMonolith500FailsClosed(t *testing.T) {
	srv := monolith(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mw := newMW(t, srv.URL, nil)

	rr, called := doRequest(t, mw, "trip2g_token=abc")

	if called {
		t.Fatal("next must not be called when monolith errors")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (fail-closed), got %d", rr.Code)
	}
}

func TestMonolithGraphQLErrorFailsClosed(t *testing.T) {
	srv := monolith(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	})
	mw := newMW(t, srv.URL, nil)

	rr, called := doRequest(t, mw, "trip2g_token=abc")

	if called {
		t.Fatal("next must not be called on graphql error")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMonolithUnreachableFailsClosed(t *testing.T) {
	// Point at a closed port; the dial fails immediately.
	mw := newMW(t, "http://127.0.0.1:1", nil)

	rr, called := doRequest(t, mw, "trip2g_token=abc")

	if called {
		t.Fatal("next must not be called when monolith is unreachable")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (fail-closed), got %d", rr.Code)
	}
}

func TestMonolithTimeoutFailsClosed(t *testing.T) {
	srv := monolith(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{"data":{"viewer":{"role":"ADMIN"}}}`))
	})
	mw := newMW(t, srv.URL, &http.Client{Timeout: 20 * time.Millisecond})

	rr, called := doRequest(t, mw, "trip2g_token=abc")

	if called {
		t.Fatal("next must not be called on timeout")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (fail-closed), got %d", rr.Code)
	}
}

func TestForwardsFullCookieHeaderVerbatim(t *testing.T) {
	var gotCookie string
	srv := monolith(t, &gotCookie, roleResponder("ADMIN"))
	mw := newMW(t, srv.URL, nil)

	const cookie = "trip2g_token=eyJ.header.sig; theme=dark; other=xyz"
	_, called := doRequest(t, mw, cookie)

	if !called {
		t.Fatal("expected admin to pass")
	}
	if gotCookie != cookie {
		t.Fatalf("cookie not forwarded verbatim:\n got:  %q\n want: %q", gotCookie, cookie)
	}
	// Sanity: no service token/HAT smuggled in.
	if strings.Contains(gotCookie, "Bearer") {
		t.Fatal("must not send a bearer/service credential")
	}
}

func TestMissingBaseURLIsError(t *testing.T) {
	if _, err := delegatedadmin.New(delegatedadmin.Config{}); err == nil {
		t.Fatal("expected error for empty MonolithBaseURL")
	}
}
