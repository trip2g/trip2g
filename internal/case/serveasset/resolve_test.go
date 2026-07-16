package serveasset_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"trip2g/internal/appreq"
	"trip2g/internal/assetindex"
	"trip2g/internal/case/serveasset"
	"trip2g/internal/db"
	"trip2g/internal/logger"
	"trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const (
	testHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testBody = "0123456789"
)

func testAsset() db.NoteAsset {
	return db.NoteAsset{ID: 7, FileName: "pic.png", Sha256Hash: testHash, Size: int64(len(testBody))}
}

type envOpts struct {
	rows        []db.NoteAsset
	ownership   assetindex.Ownership
	known       bool
	canRead     bool
	validAPIKey bool
	apiKeyErr   error
}

func newEnv(o envOpts) *EnvMock {
	return &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		NoteAssetsBySha256HashFunc: func(ctx context.Context, hash string) ([]db.NoteAsset, error) {
			return o.rows, nil
		},
		AssetOwnershipFunc: func(hash, fileName string) (assetindex.Ownership, bool) {
			return o.ownership, o.known
		},
		CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) {
			return o.canRead, nil
		},
		StreamAssetObjectFunc: func(ctx context.Context, asset db.NoteAsset, offset, length int64) (io.ReadCloser, error) {
			if offset < 0 || offset+length > int64(len(testBody)) {
				panic("stream range out of bounds")
			}
			return io.NopCloser(strings.NewReader(testBody[offset : offset+length])), nil
		},
		ValidAPIKeyFunc: func(ctx context.Context, plainKey string) (bool, error) {
			return o.validAPIKey, o.apiKeyErr
		},
	}
}

type reqOpts struct {
	path        string
	method      string
	token       *usertoken.Data // nil = anonymous
	ifNoneMatch string
	rangeHdr    string
	apiKey      string
}

func doRequest(t *testing.T, env serveasset.Env, o reqOpts) (*fasthttp.RequestCtx, bool) {
	t.Helper()

	if o.method == "" {
		o.method = http.MethodGet
	}

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(o.method)
	ctx.Request.SetRequestURI(o.path)
	if o.ifNoneMatch != "" {
		ctx.Request.Header.Set("If-None-Match", o.ifNoneMatch)
	}
	if o.rangeHdr != "" {
		ctx.Request.Header.Set("Range", o.rangeHdr)
	}
	if o.apiKey != "" {
		ctx.Request.Header.Set("X-API-Key", o.apiKey)
	}

	req := &appreq.Request{
		Env:  env,
		Req:  ctx,
		Path: o.path,
	}
	req.SetUserToken(o.token) // preset so UserToken() never hits a TokenManager
	req.StoreInContext()

	handled := serveasset.Handle(req)
	return ctx, handled
}

func body(ctx *fasthttp.RequestCtx) string {
	// SetBodyStream bodies are not materialized in Response.Body(); read the stream.
	if stream := ctx.Response.BodyStream(); stream != nil {
		b, err := io.ReadAll(stream)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
	return string(ctx.Response.Body())
}

func publicOwnership() assetindex.Ownership {
	return assetindex.Ownership{Public: true, Notes: []*model.NoteView{{Path: "free.md", Free: true}}}
}

func privateOwnership() assetindex.Ownership {
	return assetindex.Ownership{Public: false, Notes: []*model.NoteView{{Path: "paid.md"}}}
}

func TestHandle_NonAssetPathNotHandled(t *testing.T) {
	env := newEnv(envOpts{})
	_, handled := doRequest(t, env, reqOpts{path: "/some/note"})
	require.False(t, handled)
}

func TestHandle_PublicAsset_AnonymousGets200Immutable(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "pic.png")})

	require.True(t, handled)
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, "public, max-age=31536000, immutable", string(ctx.Response.Header.Peek("Cache-Control")))
	require.Equal(t, `"`+testHash+`"`, string(ctx.Response.Header.Peek("ETag")))
	require.Equal(t, "nosniff", string(ctx.Response.Header.Peek("X-Content-Type-Options")))
	require.Equal(t, "image/png", string(ctx.Response.Header.ContentType()))
	require.Equal(t, testBody, body(ctx))
	// No session lookup or per-note check needed on the public path.
	require.Empty(t, env.CanReadNoteCalls())
	require.Empty(t, env.ValidAPIKeyCalls())
}

func TestHandle_PrivateAsset_ValidAPIKey200(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true, validAPIKey: true})

	ctx, handled := doRequest(t, env, reqOpts{
		path:   model.NoteAssetURLPath(testHash, "pic.png"),
		apiKey: "valid-key",
	})

	require.True(t, handled)
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, testBody, body(ctx))
	// A valid API key already has firehose read access (pushNotes bypassACL);
	// it must not go through the per-note ACL check.
	require.Empty(t, env.CanReadNoteCalls())
}

func TestHandle_PrivateAsset_InvalidAPIKey401(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true, validAPIKey: false})

	ctx, handled := doRequest(t, env, reqOpts{
		path:   model.NoteAssetURLPath(testHash, "pic.png"),
		apiKey: "bogus-key",
	})

	require.True(t, handled)
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode(),
		"an invalid API key must fail closed, not fall through to anonymous")
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_PrivateAsset_APIKeyDBError500(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true, apiKeyErr: errors.New("db exploded")})

	ctx, handled := doRequest(t, env, reqOpts{
		path:   model.NoteAssetURLPath(testHash, "pic.png"),
		apiKey: "some-key",
	})

	require.True(t, handled)
	require.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_IfNoneMatch_Returns304(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{
		path:        model.NoteAssetURLPath(testHash, "pic.png"),
		ifNoneMatch: `"` + testHash + `"`,
	})

	require.True(t, handled)
	require.Equal(t, http.StatusNotModified, ctx.Response.StatusCode())
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_PrivateAsset_NoSession401(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "pic.png")})

	require.True(t, handled)
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode())
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_PrivateAsset_WrongUser403(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true, canRead: false})

	ctx, handled := doRequest(t, env, reqOpts{
		path:  model.NoteAssetURLPath(testHash, "pic.png"),
		token: &usertoken.Data{ID: 2, Role: "user"},
	})

	require.True(t, handled)
	require.Equal(t, http.StatusForbidden, ctx.Response.StatusCode())
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_PrivateAsset_AuthorizedUser200Private(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: privateOwnership(), known: true, canRead: true})

	ctx, handled := doRequest(t, env, reqOpts{
		path:  model.NoteAssetURLPath(testHash, "pic.png"),
		token: &usertoken.Data{ID: 2, Role: "user"},
	})

	require.True(t, handled)
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	require.Equal(t, "private, max-age=300", string(ctx.Response.Header.Peek("Cache-Control")))
	require.Equal(t, testBody, body(ctx))
}

func TestHandle_UnknownOwnership_FailsClosed(t *testing.T) {
	// Hash exists in the DB but is not referenced by any live note or layout:
	// anonymous → 401, regular user → 403, admin → 200.
	rows := []db.NoteAsset{testAsset()}

	ctx, _ := doRequest(t, newEnv(envOpts{rows: rows, known: false}), reqOpts{
		path: model.NoteAssetURLPath(testHash, "pic.png"),
	})
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode())

	ctx, _ = doRequest(t, newEnv(envOpts{rows: rows, known: false, canRead: true}), reqOpts{
		path:  model.NoteAssetURLPath(testHash, "pic.png"),
		token: &usertoken.Data{ID: 2, Role: "user"},
	})
	require.Equal(t, http.StatusForbidden, ctx.Response.StatusCode(),
		"no owning notes to grant access through — must fail closed")

	ctx, _ = doRequest(t, newEnv(envOpts{rows: rows, known: false}), reqOpts{
		path:  model.NoteAssetURLPath(testHash, "pic.png"),
		token: &usertoken.Data{ID: 1, Role: "admin"},
	})
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
}

func TestHandle_UnknownHash404(t *testing.T) {
	env := newEnv(envOpts{rows: nil, known: false})

	ctx, handled := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "pic.png")})

	require.True(t, handled)
	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
}

func TestHandle_MalformedHash404(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	for _, path := range []string{
		model.AssetRoutePrefix + "shorthash/pic.png",
		model.AssetRoutePrefix + strings.Repeat("z", 64) + "/pic.png",
		model.AssetRoutePrefix + testHash, // no fileName segment
		model.AssetRoutePrefix + testHash + "/",
	} {
		ctx, handled := doRequest(t, env, reqOpts{path: path})
		require.True(t, handled, path)
		require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode(), path)
	}
}

func TestHandle_FileNameMismatch404(t *testing.T) {
	// Correct hash but a fileName that matches no stored row must 404 — it
	// would otherwise let an attacker pick the served Content-Type.
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "evil.html")})

	require.True(t, handled)
	require.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
	require.Empty(t, env.StreamAssetObjectCalls())
}

func TestHandle_RangeRequest206(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{
		path:     model.NoteAssetURLPath(testHash, "pic.png"),
		rangeHdr: "bytes=2-5",
	})

	require.True(t, handled)
	require.Equal(t, http.StatusPartialContent, ctx.Response.StatusCode())
	require.Equal(t, "bytes 2-5/10", string(ctx.Response.Header.Peek("Content-Range")))
	require.Equal(t, "2345", body(ctx))
}

func TestHandle_RangeVariants(t *testing.T) {
	tests := []struct {
		name      string
		rangeHdr  string
		status    int
		wantBody  string
		wantRange string
	}{
		{name: "open ended", rangeHdr: "bytes=7-", status: 206, wantBody: "789", wantRange: "bytes 7-9/10"},
		{name: "suffix", rangeHdr: "bytes=-3", status: 206, wantBody: "789", wantRange: "bytes 7-9/10"},
		{name: "end clamped", rangeHdr: "bytes=8-99", status: 206, wantBody: "89", wantRange: "bytes 8-9/10"},
		{name: "start beyond size", rangeHdr: "bytes=10-", status: 416},
		{name: "inverted", rangeHdr: "bytes=5-2", status: 416},
		{name: "multi range ignored", rangeHdr: "bytes=0-1,3-4", status: 200, wantBody: testBody},
		{name: "garbage ignored", rangeHdr: "bytes=abc", status: 200, wantBody: testBody},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})
			ctx, _ := doRequest(t, env, reqOpts{
				path:     model.NoteAssetURLPath(testHash, "pic.png"),
				rangeHdr: tt.rangeHdr,
			})
			require.Equal(t, tt.status, ctx.Response.StatusCode())
			if tt.wantRange != "" {
				require.Equal(t, tt.wantRange, string(ctx.Response.Header.Peek("Content-Range")))
			}
			if tt.status != http.StatusRequestedRangeNotSatisfiable {
				require.Equal(t, tt.wantBody, body(ctx))
			}
		})
	}
}

func TestHandle_MethodNotAllowed(t *testing.T) {
	env := newEnv(envOpts{rows: []db.NoteAsset{testAsset()}, ownership: publicOwnership(), known: true})

	ctx, handled := doRequest(t, env, reqOpts{
		path:   model.NoteAssetURLPath(testHash, "pic.png"),
		method: http.MethodPost,
	})

	require.True(t, handled)
	require.Equal(t, http.StatusMethodNotAllowed, ctx.Response.StatusCode())
}

func TestHandle_MultipleRowsSameHash_PicksMatchingFileName(t *testing.T) {
	other := db.NoteAsset{ID: 9, FileName: "copy.png", Sha256Hash: testHash, Size: int64(len(testBody))}
	env := newEnv(envOpts{rows: []db.NoteAsset{other, testAsset()}, ownership: publicOwnership(), known: true})

	ctx, _ := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "pic.png")})

	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())
	calls := env.StreamAssetObjectCalls()
	require.Len(t, calls, 1)
	require.Equal(t, int64(7), calls[0].Asset.ID)
}

// TestHandle_SharedHashDifferentFileName_PrivateRowNotLeakedViaPublicSibling
// pins the fix for a hash-only-keyed publicness leak: a private row
// ("private.png") shares its sha256 with an unrelated public row
// ("public.png", different note_assets row, identical bytes). Ownership must
// be looked up per (hash, fileName) — never per hash alone — so fetching the
// private row's own filename anonymously must still be denied even though
// the same hash is public under a different filename.
func TestHandle_SharedHashDifferentFileName_PrivateRowNotLeakedViaPublicSibling(t *testing.T) {
	publicRow := db.NoteAsset{ID: 1, FileName: "public.png", Sha256Hash: testHash, Size: int64(len(testBody))}
	privateRow := db.NoteAsset{ID: 2, FileName: "private.png", Sha256Hash: testHash, Size: int64(len(testBody))}
	rows := []db.NoteAsset{publicRow, privateRow}

	env := &EnvMock{
		LoggerFunc: func() logger.Logger { return &logger.DummyLogger{} },
		NoteAssetsBySha256HashFunc: func(ctx context.Context, hash string) ([]db.NoteAsset, error) {
			return rows, nil
		},
		AssetOwnershipFunc: func(hash, fileName string) (assetindex.Ownership, bool) {
			switch fileName {
			case "public.png":
				return publicOwnership(), true
			case "private.png":
				return privateOwnership(), true
			default:
				return assetindex.Ownership{}, false
			}
		},
		CanReadNoteFunc: func(ctx context.Context, note *model.NoteView) (bool, error) { return false, nil },
		StreamAssetObjectFunc: func(ctx context.Context, asset db.NoteAsset, offset, length int64) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(testBody[offset : offset+length])), nil
		},
	}

	// The public sibling is fetchable anonymously.
	ctx, _ := doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "public.png")})
	require.Equal(t, http.StatusOK, ctx.Response.StatusCode())

	// The private row, same hash, must NOT be fetchable anonymously just
	// because a sibling row with the same bytes is public.
	ctx, _ = doRequest(t, env, reqOpts{path: model.NoteAssetURLPath(testHash, "private.png")})
	require.Equal(t, http.StatusUnauthorized, ctx.Response.StatusCode(),
		"a private row must not inherit publicness from a same-hash sibling row")
}
