package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// testPersonalTokenResolver implements appreq.PersonalTokenResolver for tests.
type testPersonalTokenResolver struct {
	token *usertoken.Data
	err   error
	calls int
}

func (r *testPersonalTokenResolver) Resolve(_ context.Context, _ string) (*usertoken.Data, error) {
	r.calls++
	return r.token, r.err
}

// buildMCPFasthttpCtx builds a *fasthttp.RequestCtx with the given body and Authorization header.
func buildMCPFasthttpCtx(body []byte, authHeader string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	// Wire the fake server so ctx works as a context.Context (Done() panics on a bare RequestCtx).
	ctx.Init2(nil, nil, true)
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/_system/mcp")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody(body)
	if authHeader != "" {
		ctx.Request.Header.Set("Authorization", authHeader)
	}
	return ctx
}

// mcpInitBody is a valid MCP initialize request.
var mcpInitBody = func() []byte { //nolint:gochecknoglobals // test package global
	b, _ := json.Marshal(mcp.Request{JSONRPC: "2.0", Method: "initialize", ID: 1})
	return b
}()

// buildDispatchEnv returns a minimal EnvMock for endpoint dispatch tests.
// verifyInboundWillFail controls whether FederationSecretByKID panics (to detect unexpected calls).
func buildDispatchEnv(t *testing.T, verifyInboundWillFail bool) *EnvMock {
	t.Helper()
	env := &EnvMock{
		LatestNoteViewsFunc: func() *appmodel.NoteViews {
			return &appmodel.NoteViews{List: []*appmodel.NoteView{}, PathMap: map[string]*appmodel.NoteView{}}
		},
		LoggerFunc:                  func() logger.Logger { return &logger.DummyLogger{} },
		FeaturesFunc:                func() features.Features { return features.Features{} },
		CanReadNoteFunc:             func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil },
		FederatedGraphQLEnabledFunc: func() bool { return false },
		MCPMetricsFunc:              func() *metrics.MCPMetrics { return nil },
		SiteConfigFunc:              func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },

		FederatedFanoutConcurrencyFunc: func() int { return 5 },
		FederatedFanoutLimitFunc:       func() int { return 7 },
		FederatedFanoutTimeoutFunc:     func() time.Duration { return 5 * time.Second },
	}
	if verifyInboundWillFail {
		env.FederationSecretByKIDFunc = func(_ context.Context, _ string) (db.FederationSecret, bool, error) {
			t.Error("verifyInbound was called — must NOT be called for a t2g_* personal token")
			return db.FederationSecret{}, false, errors.New("verifyInbound should not be called")
		}
	}
	return env
}

// noopTokenManager is a minimal usertoken.Manager with a distinct cookie name that
// will never match any test request, so Extract always returns ErrTokenMissing.
var noopTokenManager = usertoken.NewManager(usertoken.Config{ //nolint:gochecknoglobals // test package global
	CookieName: "__noop_test_cookie__",
	Secret:     "test-secret-32-bytes-long-filler!",
})

// wiredRequest creates an appreq.Request backed by the given fasthttp context,
// with the env and optional PersonalTokenResolver set.
func wiredRequest(fasthttpCtx *fasthttp.RequestCtx, env interface{}, resolver appreq.PersonalTokenResolver) *appreq.Request {
	req := appreq.Acquire()
	req.Env = env
	req.Req = fasthttpCtx
	req.TokenManager = noopTokenManager
	req.PersonalTokenResolver = resolver
	req.StoreInContext()
	return req
}

// TestPersonalTokenBearerSkipsVerifyInbound: t2g_* Bearer must NOT call verifyInbound.
// The resolver returns a valid user. verifyInbound is wired to fail if called.
func TestPersonalTokenBearerSkipsVerifyInbound(t *testing.T) {
	resolver := &testPersonalTokenResolver{token: &usertoken.Data{ID: 7, Role: "user"}}
	env := buildDispatchEnv(t, true /* verifyInbound would fail if called */)

	fasthttpCtx := buildMCPFasthttpCtx(mcpInitBody, "Bearer t2g_sometokenvalue")
	req := wiredRequest(fasthttpCtx, env, resolver)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error, "expected success, got: %v", resp.Error)
}

// TestFederationBearerStillUsesVerifyInbound: non-t2g_ Bearer goes through verifyInbound.
// An invalid federation JWT should return a federation auth error in the JSON-RPC response.
func TestFederationBearerStillUsesVerifyInbound(t *testing.T) {
	// Three base64url segments = federation JWT format; content deliberately invalid.
	federationToken := "aGVhZGVy.Y2xhaW1z.c2ln"

	env := buildDispatchEnv(t, false)
	// Wire federation secret lookup to return not-found → ErrFedAuthUnknownKid.
	env.FederationSecretByKIDFunc = func(_ context.Context, _ string) (db.FederationSecret, bool, error) {
		return db.FederationSecret{}, false, nil
	}
	env.DecryptDataFunc = func(b []byte) ([]byte, error) { return b, nil }
	env.ListFederationSecretSubgraphsByKIDFunc = func(_ context.Context, _ string) ([]string, error) {
		return nil, nil
	}

	fasthttpCtx := buildMCPFasthttpCtx(mcpInitBody, "Bearer "+federationToken)
	req := wiredRequest(fasthttpCtx, env, nil /* no personal resolver needed */)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err) // Handle itself returns no error; auth failure is in JSON-RPC response

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.NotNil(t, resp.Error)
	require.Contains(t, resp.Error.Message, "Federation auth failed")
}

// TestPersonalTokenAdminSeesAllKBNotes: admin personal token → accessibleKBNotes returns all KB notes.
func TestPersonalTokenAdminSeesAllKBNotes(t *testing.T) {
	kbNote := &appmodel.NoteView{
		PathID:             10,
		MCPFederationKBURL: "https://peer.example/_system/mcp",
		MCPFederationKBID:  "peer",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	resolver := &testPersonalTokenResolver{token: &usertoken.Data{ID: 1, Role: "admin"}}

	callBody, _ := json.Marshal(mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  mustMarshalRaw(mcp.CallToolParams{Name: "federated_search", Arguments: json.RawMessage(`{"query":"x"}`)}),
		ID:      2,
	})

	var federationQueriedKBIDs []string
	env := buildDispatchEnv(t, true)
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews { return nvs }
	env.CanReadNoteFunc = func(_ context.Context, _ *appmodel.NoteView) (bool, error) { return true, nil }
	env.FederationClientFunc = func(_ context.Context, kbID string) (appmodel.Federation, error) {
		federationQueriedKBIDs = append(federationQueriedKBIDs, kbID)
		return &federationMock{
			searchFunc: func(_ context.Context, params appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
				return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "ok"}}}, nil
			},
		}, nil
	}

	fasthttpCtx := buildMCPFasthttpCtx(callBody, "Bearer t2g_admintoken")
	req := wiredRequest(fasthttpCtx, env, resolver)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error)
	require.Equal(t, []string{"peer"}, federationQueriedKBIDs)
}

// TestPersonalTokenNonAdminKBNoteFiltered: non-admin personal token → only accessible KB notes queried.
func TestPersonalTokenNonAdminKBNoteFiltered(t *testing.T) {
	allowedKB := &appmodel.NoteView{PathID: 11, MCPFederationKBURL: "https://a.example/_system/mcp", MCPFederationKBID: "a"}
	deniedKB := &appmodel.NoteView{PathID: 12, MCPFederationKBURL: "https://b.example/_system/mcp", MCPFederationKBID: "b"}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{
		appmodel.NewMCPFederationNote(allowedKB),
		appmodel.NewMCPFederationNote(deniedKB),
	}

	resolver := &testPersonalTokenResolver{token: &usertoken.Data{ID: 99, Role: "user"}}

	callBody, _ := json.Marshal(mcp.Request{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  mustMarshalRaw(mcp.CallToolParams{Name: "federated_search", Arguments: json.RawMessage(`{"query":"x"}`)}),
		ID:      3,
	})

	var queriedKBIDs []string
	env := buildDispatchEnv(t, true)
	env.LatestNoteViewsFunc = func() *appmodel.NoteViews { return nvs }
	// User can only read allowedKB (pathID 11).
	env.CanReadNoteFunc = func(_ context.Context, note *appmodel.NoteView) (bool, error) {
		return note.PathID == allowedKB.PathID, nil
	}
	env.FederationClientFunc = func(_ context.Context, kbID string) (appmodel.Federation, error) {
		queriedKBIDs = append(queriedKBIDs, kbID)
		return &federationMock{
			searchFunc: func(_ context.Context, _ appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
				return appmodel.FederationResult{Content: []appmodel.FederationContent{{Type: "text", Text: "ok"}}}, nil
			},
		}, nil
	}

	fasthttpCtx := buildMCPFasthttpCtx(callBody, "Bearer t2g_usertoken")
	req := wiredRequest(fasthttpCtx, env, resolver)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.Nil(t, resp.Error)
	require.Equal(t, []string{"a"}, queriedKBIDs, "non-admin should only query KB notes they can access")
}

func TestMCPEndpointDepthEnforcement(t *testing.T) {
	buildEnvWithMaxDepth := func(t *testing.T, maxDepth int) *EnvMock {
		t.Helper()
		env := buildDispatchEnv(t, false)
		env.FederationMaxDepthFunc = func() int { return maxDepth }
		return env
	}

	buildCtxWithDepth := func(body []byte, depthHeader string) *fasthttp.RequestCtx {
		ctx := buildMCPFasthttpCtx(body, "")
		if depthHeader != "" {
			ctx.Request.Header.Set("X-MCP-Federation-Depth", depthHeader)
		}
		return ctx
	}

	t.Run("no depth header passes through", func(t *testing.T) {
		env := buildEnvWithMaxDepth(t, 3)
		fasthttpCtx := buildCtxWithDepth(mcpInitBody, "")
		req := wiredRequest(fasthttpCtx, env, nil)
		defer appreq.Release(req)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
		require.Nil(t, resp.Error)
	})

	t.Run("depth below max passes through", func(t *testing.T) {
		env := buildEnvWithMaxDepth(t, 3)
		fasthttpCtx := buildCtxWithDepth(mcpInitBody, "2")
		req := wiredRequest(fasthttpCtx, env, nil)
		defer appreq.Release(req)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
		require.Nil(t, resp.Error)
	})

	t.Run("depth equal to max is rejected", func(t *testing.T) {
		env := buildEnvWithMaxDepth(t, 3)
		fasthttpCtx := buildCtxWithDepth(mcpInitBody, "3")
		req := wiredRequest(fasthttpCtx, env, nil)
		defer appreq.Release(req)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
		require.NotNil(t, resp.Error)
		require.Contains(t, resp.Error.Message, "max depth")
	})

	t.Run("depth above max is rejected", func(t *testing.T) {
		env := buildEnvWithMaxDepth(t, 3)
		fasthttpCtx := buildCtxWithDepth(mcpInitBody, "10")
		req := wiredRequest(fasthttpCtx, env, nil)
		defer appreq.Release(req)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
		require.NotNil(t, resp.Error)
		require.Contains(t, resp.Error.Message, "max depth")
	})
}

func TestDispatch_APIKeyAuth_AdminTools(t *testing.T) {
	env := buildDispatchEnv(t, true) // verifyInbound must NOT be called
	adminTools := true
	env.ResolveAPIKeyFunc = func(_ context.Context, value, action string) (*db.ApiKey, error) {
		require.Equal(t, "test-api-key", value)
		require.Equal(t, "mcp", action)
		return &db.ApiKey{ID: 1, EnableMcpAdminTools: &adminTools}, nil
	}

	fasthttpCtx := buildMCPFasthttpCtx(mcpInitBody, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "test-api-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)
	require.Len(t, env.ResolveAPIKeyCalls(), 1)
}

func TestDispatch_APIKeyAuth_InvalidKey(t *testing.T) {
	env := buildDispatchEnv(t, true) // verifyInbound must NOT be called
	env.ResolveAPIKeyFunc = func(_ context.Context, _, _ string) (*db.ApiKey, error) {
		return nil, errors.New("invalid API key")
	}

	fasthttpCtx := buildMCPFasthttpCtx(mcpInitBody, "")
	fasthttpCtx.Request.Header.Set("X-API-Key", "bad-key")

	req := wiredRequest(fasthttpCtx, env, nil)
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err) // error in JSON body, not Go error

	var resp mcp.Response
	require.NoError(t, json.Unmarshal(fasthttpCtx.Response.Body(), &resp))
	require.NotNil(t, resp.Error)
	require.Contains(t, resp.Error.Message, "Auth failed")
}

func mustMarshalRaw(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
