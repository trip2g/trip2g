package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"trip2g/internal/appreq"
	"trip2g/internal/shortapitoken"
)

const testSecret = "test-secret-for-stamp"

func newBearerRequest(t *testing.T, token string) *appreq.Request {
	t.Helper()
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.Set("Authorization", "Bearer "+token)
	fctx.Request.SetRequestURI("http://example.com/graphql")
	req := &appreq.Request{Req: fctx}
	return req
}

func TestStampShortAPIToken(t *testing.T) {
	scopedToken := func(d shortapitoken.Data, ttl time.Duration) string {
		tok, err := shortapitoken.Sign(d, testSecret, ttl)
		require.NoError(t, err)
		return tok
	}

	tests := []struct {
		name          string
		setup         func(t *testing.T) *appreq.Request
		wantReadPats  []string
		wantWritePats []string
		wantDepth     int
		wantKind      string
		wantID        int64
		wantScoped    bool
	}{
		{
			name: "scoped token stamps read and write patterns",
			setup: func(t *testing.T) *appreq.Request {
				tok := scopedToken(shortapitoken.Data{
					Depth:         2,
					ReadPatterns:  []string{"boards/**", "docs/**"},
					WritePatterns: []string{"boards/**"},
					DeliveryKind:  "change",
					DeliveryID:    77,
				}, time.Hour)
				return newBearerRequest(t, tok)
			},
			wantReadPats:  []string{"boards/**", "docs/**"},
			wantWritePats: []string{"boards/**"},
			wantDepth:     2,
			wantKind:      "change",
			wantID:        77,
			wantScoped:    true,
		},
		{
			name: "personal token (t2g_) is not stamped",
			setup: func(t *testing.T) *appreq.Request {
				return newBearerRequest(t, "t2g_somevalue")
			},
		},
		{
			name: "no Authorization header — no-op",
			setup: func(t *testing.T) *appreq.Request {
				fctx := &fasthttp.RequestCtx{}
				fctx.Request.SetRequestURI("http://example.com/graphql")
				return &appreq.Request{Req: fctx}
			},
		},
		{
			name: "invalid JWT — no-op (anonymous)",
			setup: func(t *testing.T) *appreq.Request {
				return newBearerRequest(t, "not.a.valid.jwt")
			},
		},
		{
			name: "expired token — no-op (anonymous)",
			setup: func(t *testing.T) *appreq.Request {
				tok := scopedToken(shortapitoken.Data{
					ReadPatterns: []string{"boards/**"},
				}, -time.Hour)
				return newBearerRequest(t, tok)
			},
		},
		{
			name: "already-stamped req — not overwritten (idempotent)",
			setup: func(t *testing.T) *appreq.Request {
				tok := scopedToken(shortapitoken.Data{
					ReadPatterns: []string{"new/**"},
				}, time.Hour)
				req := newBearerRequest(t, tok)
				req.WebhookReadPatterns = []string{"old/**"}
				return req
			},
			wantReadPats: []string{"old/**"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setup(t)
			stampShortAPIToken(req, testSecret)

			require.Equal(t, tt.wantReadPats, req.WebhookReadPatterns)
			require.Equal(t, tt.wantWritePats, req.WebhookWritePatterns)
			require.Equal(t, tt.wantDepth, req.WebhookDepth)
			require.Equal(t, tt.wantKind, req.WebhookDeliveryKind)
			require.Equal(t, tt.wantID, req.WebhookDeliveryID)
			require.Equal(t, tt.wantScoped, req.WebhookScoped)
		})
	}
}

// TestStampShortAPIToken_NoteResolverScope verifies the end-to-end contract that
// stampShortAPIToken (called from makeAroundOperations) stamps read patterns so
// that the Note/search resolvers can filter by path. This is the F1 regression.
//
// Pre-fix: stampShortAPIToken was never called for QUERY operations so
// req.WebhookReadPatterns was always nil, and the path check in the Note
// resolver was dead for the scoped-token path.
func TestStampShortAPIToken_ScopedTokenPath(t *testing.T) {
	tok, err := shortapitoken.Sign(shortapitoken.Data{
		Depth:         1,
		ReadPatterns:  []string{"boards/**"},
		WritePatterns: []string{"boards/**"},
		DeliveryKind:  "change",
		DeliveryID:    1,
	}, testSecret, time.Hour)
	require.NoError(t, err)

	req := newBearerRequest(t, tok)

	// Before the fix, stampShortAPIToken was never called on the QUERY path,
	// so WebhookReadPatterns remained nil.
	// After the fix, calling stampShortAPIToken (as makeAroundOperations now does)
	// populates WebhookReadPatterns.
	stampShortAPIToken(req, testSecret)

	require.NotEmpty(t, req.WebhookReadPatterns, "read patterns must be stamped for scoped token")
	require.Equal(t, []string{"boards/**"}, req.WebhookReadPatterns)
	require.Equal(t, 1, req.WebhookDepth)
}
