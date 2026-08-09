package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"trip2g/internal/appreq"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"
	"trip2g/internal/ptr"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// updateGolden rewrites testdata/endpoint_golden.txt instead of comparing.
var updateGolden = flag.Bool("update-golden", false, "rewrite the MCP endpoint golden file") //nolint:gochecknoglobals // test flag

// The golden file pins the exact bytes the MCP endpoint puts on the wire for a
// matrix of representative requests: status code, content type and response
// body. It is the regression contract for reimplementing the transport on top
// of the official Go MCP SDK — any wire change shows up as a golden diff that
// has to be reviewed and consciously accepted.

const goldenPath = "testdata/endpoint_golden.txt"

// goldenCase is one request pushed through the real endpoint.
type goldenCase struct {
	name     string
	body     string
	query    string            // raw query string, without '?'
	headers  map[string]string // extra request headers
	noAccept bool              // omit the Accept header a spec-compliant client sends
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{name: "initialize", body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"golden","version":"1"}}}`},
		{name: "initialize_no_params", body: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`},
		{name: "initialize_method_override", body: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, query: "method=guide"},
		{name: "initialize_method_override_unknown", body: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, query: "method=nope"},
		{name: "initialize_method_override_private", body: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, query: "method=private-tool"},
		{name: "notifications_initialized", body: `{"jsonrpc":"2.0","id":null,"method":"notifications/initialized"}`},

		{name: "tools_list_anonymous", body: `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`},
		{name: "tools_list_admin_api_key", body: `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, headers: map[string]string{"X-API-Key": "admin-key"}},
		{name: "tools_list_plain_api_key", body: `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, headers: map[string]string{"X-API-Key": "plain-key"}},

		{name: "call_search", body: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"alpha"}}}`},
		{name: "call_search_limits", body: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"alpha","limit":99,"detail_limit":1}}}`},
		{name: "call_search_missing_query", body: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{}}}`},
		{name: "call_search_wrong_arg_type", body: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":42}}}`},

		{name: "call_note_html_by_path", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"path":"doc.md"}}}`},
		{name: "call_note_html_by_pid_number", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"pid":10}}}`},
		{name: "call_note_html_by_pid_string", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"pid":"10"}}}`},
		{name: "call_note_html_chunk_ref_in_pid", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"pid":"p10:c2"}}}`},
		{name: "call_note_html_toc_path", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"path":"doc.md","toc_path":["Beta"]}}}`},
		{name: "call_note_html_toc_path_miss", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"path":"doc.md","toc_path":["Nope"]}}}`},
		{name: "call_note_html_no_selector", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{}}}`},
		{name: "call_note_html_not_found", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"path":"missing.md"}}}`},
		{name: "call_note_html_private", body: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"note_html","arguments":{"path":"private.md"}}}`},

		{name: "call_expand_root", body: `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"expand","arguments":{"path":"doc.md"}}}`},
		{name: "call_expand_section", body: `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"expand","arguments":{"path":"doc.md","toc_path":["Beta"]}}}`},
		{name: "call_expand_no_selector", body: `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"expand","arguments":{}}}`},

		{name: "call_federated_search", body: `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"federated_search","arguments":{"query":"alpha"}}}`},
		{name: "call_federated_search_by_kb", body: `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"federated_search","arguments":{"query":"alpha","kb_id":"peer"}}}`},
		{name: "call_federated_search_unknown_kb", body: `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"federated_search","arguments":{"query":"alpha","kb_id":"ghost"}}}`},
		{name: "call_federated_instructions", body: `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"federated_instructions","arguments":{"kb_id":"peer"}}}`},

		{name: "call_graphql_request_unauthorized", body: `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"graphql_request","arguments":{"query":"{admin{__typename}}"}}}`},
		{name: "call_graphql_introspection_unauthorized", body: `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"graphql_introspection","arguments":{"pattern":"Note"}}}`},
		{name: "call_graphql_request_admin", body: `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"graphql_request","arguments":{"query":"{admin{__typename}}"}}}`, headers: map[string]string{"X-API-Key": "admin-key"}},

		{name: "call_dynamic_tool", body: `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"code-review","arguments":{}}}`},
		{name: "call_dynamic_tool_private", body: `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"private-tool","arguments":{}}}`},
		{name: "call_unknown_tool", body: `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"does-not-exist","arguments":{}}}`},
		{name: "call_missing_params", body: `{"jsonrpc":"2.0","id":8,"method":"tools/call"}`},

		{name: "unknown_method", body: `{"jsonrpc":"2.0","id":9,"method":"resources/list","params":{}}`},
		{name: "unimplemented_method", body: `{"jsonrpc":"2.0","id":9,"method":"totally/unknown","params":{}}`},
		{name: "bad_jsonrpc_version", body: `{"jsonrpc":"1.0","id":9,"method":"tools/list","params":{}}`},
		{name: "parse_error", body: `{not json`},
		{name: "string_id", body: `{"jsonrpc":"2.0","id":"abc","method":"tools/list","params":{}}`},

		{name: "federation_depth_ok", body: `{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}`, headers: map[string]string{"X-MCP-Federation-Depth": "2"}},
		{name: "federation_depth_exceeded", body: `{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}`, headers: map[string]string{"X-MCP-Federation-Depth": "9"}},
		{name: "federation_depth_malformed", body: `{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}`, headers: map[string]string{"X-MCP-Federation-Depth": "abc"}},

		{name: "bad_api_key", body: `{"jsonrpc":"2.0","id":11,"method":"tools/list","params":{}}`, headers: map[string]string{"X-API-Key": "nope"}},
		{name: "malformed_bearer", body: `{"jsonrpc":"2.0","id":11,"method":"tools/list","params":{}}`, headers: map[string]string{"Authorization": "Basic zzz"}},

		// Streamable HTTP requires clients to accept both media types; a client
		// that omits Accept is rejected by the transport before dispatch.
		{name: "missing_accept_header", body: `{"jsonrpc":"2.0","id":12,"method":"tools/list","params":{}}`, noAccept: true},
	}
}

// acceptHeader is what the MCP Streamable HTTP spec requires clients to send.
const acceptHeader = "application/json, text/event-stream"

func TestEndpointGolden(t *testing.T) {
	var out bytes.Buffer
	for _, c := range goldenCases() {
		status, contentType, body := runGoldenCase(t, c)
		fmt.Fprintf(&out, "### %s\nHTTP %d %s\n%s\n\n", c.name, status, contentType, body)
	}

	got := out.String()
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0o644))
		t.Log("golden file updated")
		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file missing — run: go test ./internal/case/mcp -run TestEndpointGolden -update-golden")
	require.Equal(t, string(want), got, "MCP wire output changed; review the diff, then re-record with -update-golden")
}

func runGoldenCase(t *testing.T, c goldenCase) (int, string, string) {
	t.Helper()

	fasthttpCtx := &fasthttp.RequestCtx{}
	fasthttpCtx.Init2(nil, nil, true)
	fasthttpCtx.Request.Header.SetMethod("POST")
	uri := "/_system/mcp"
	if c.query != "" {
		uri += "?" + c.query
	}
	fasthttpCtx.Request.SetRequestURI(uri)
	fasthttpCtx.Request.Header.SetContentType("application/json")
	if !c.noAccept {
		fasthttpCtx.Request.Header.Set("Accept", acceptHeader)
	}
	fasthttpCtx.Request.SetBodyString(c.body)
	for k, v := range c.headers {
		fasthttpCtx.Request.Header.Set(k, v)
	}

	req := goldenRequest(fasthttpCtx, goldenEnv())
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	return fasthttpCtx.Response.StatusCode(),
		string(fasthttpCtx.Response.Header.ContentType()),
		normalizeGolden(fasthttpCtx.Response.Body())
}

// goldenTokenManager uses a cookie name no test request carries, so token
// extraction always misses and the matrix runs as an anonymous client unless a
// case sets an auth header explicitly.
var goldenTokenManager = usertoken.NewManager(usertoken.Config{ //nolint:gochecknoglobals // test package global
	CookieName: "__golden_test_cookie__",
	Secret:     "test-secret-32-bytes-long-filler!",
})

func goldenRequest(fasthttpCtx *fasthttp.RequestCtx, env interface{}) *appreq.Request {
	req := appreq.Acquire()
	req.Env = env
	req.Req = fasthttpCtx
	req.TokenManager = goldenTokenManager
	req.StoreInContext()
	return req
}

// latencyRe blanks the wall-clock latency federation results carry, which is
// the only nondeterministic field in the matrix.
var latencyRe = regexp.MustCompile(`"latency":\s*"[^"]*"`) //nolint:gochecknoglobals // compiled once for tests

func normalizeGolden(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 {
		return "<empty>"
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		return strings.TrimRight(string(body), "\n")
	}
	return latencyRe.ReplaceAllString(pretty.String(), `"latency": "<elided>"`)
}

// goldenFedClient is a fixed federation peer: deterministic content, no network.
type goldenFedClient struct{ appmodel.Federation }

func (goldenFedClient) Search(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer search hit"), nil
}

func (goldenFedClient) FederatedSearch(context.Context, appmodel.FederationSearchParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer federated search hit"), nil
}

func (goldenFedClient) Instructions(context.Context) (appmodel.FederationResult, error) {
	return goldenFedResult("peer instructions"), nil
}

func (goldenFedClient) FederatedInstructions(context.Context, appmodel.FederationInstructionsParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer federated instructions"), nil
}

func goldenFedResult(text string) appmodel.FederationResult {
	return appmodel.FederationResult{
		Content:           []appmodel.FederationContent{{Type: "text", Text: text}},
		StructuredContent: json.RawMessage(`{"query":"alpha","results":[{"title":"Peer Note","note_id":1,"note_path":"peer.md","href":"/peer","url":"https://peer.example/peer","kind":"note","score":1}]}`),
	}
}

// goldenEnv builds the fixture corpus: a readable note with sections, a private
// note, dynamic tool notes (public and private), and one federation peer.
func goldenEnv() *EnvMock {
	doc := &appmodel.NoteView{
		Path:      "doc.md",
		PathID:    10,
		Title:     "Doc",
		Permalink: "/doc",
		HTML:      "<h2 id=\"alpha\">Alpha</h2>\n<p>alpha body</p>\n<h2 id=\"beta\">Beta</h2>\n<p>beta body</p>",
		Content:   []byte("# Doc\n\n## Alpha\n\nalpha body\n\n## Beta\n\nbeta body\n"),
		Headings: appmodel.NoteViewHeadings{
			{Text: "Alpha", Level: 2, ID: "alpha"},
			{Text: "Beta", Level: 2, ID: "beta"},
		},
	}
	private := &appmodel.NoteView{
		Path:      "private.md",
		PathID:    11,
		Title:     "Private",
		Permalink: "/private",
		HTML:      "<p>secret</p>",
	}
	initNote := &appmodel.NoteView{
		Path:      "_mcp_initialize.md",
		PathID:    12,
		MCPMethod: "initialize",
		Content:   []byte("---\nmcp_method: initialize\n---\n\nGolden server instructions."),
	}
	guide := &appmodel.NoteView{
		Path:      "_mcp_guide.md",
		PathID:    13,
		MCPMethod: "guide",
		Content:   []byte("---\nmcp_method: guide\n---\n\nGuide instructions."),
	}
	codeReview := &appmodel.NoteView{
		Path:           "_mcp_code_review.md",
		PathID:         14,
		MCPMethod:      "code-review",
		MCPDescription: "Detailed code review",
		Content:        []byte("---\nmcp_method: code-review\n---\n\nReview this code carefully."),
	}
	privateTool := &appmodel.NoteView{
		Path:      "_mcp_private.md",
		PathID:    15,
		MCPMethod: "private-tool",
		Content:   []byte("---\nmcp_method: private-tool\n---\n\nSecret tool."),
	}
	peer := &appmodel.NoteView{
		Path:               "_mcp_peer.md",
		PathID:             16,
		MCPFederationKBURL: "https://peer.example/_system/mcp",
		MCPFederationKBID:  "peer",
	}

	notes := []*appmodel.NoteView{doc, private, initNote, guide, codeReview, privateTool, peer}
	views := appmodel.NewNoteViews()
	views.List = notes
	for _, n := range notes {
		views.PathMap[n.Path] = n
		if n.Permalink != "" {
			views.Map[n.Permalink] = n
		}
	}
	views.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(peer)}

	searchHits := []appmodel.SearchResult{{
		NoteView:           doc,
		URL:                doc.Permalink,
		Score:              0.9,
		HighlightedContent: []string{"alpha body"},
	}}

	return &EnvMock{
		LatestNoteViewsFunc:  func() *appmodel.NoteViews { return views },
		LiveNoteViewsFunc:    func() *appmodel.NoteViews { return views },
		LatestNoteChunksFunc: func() []appmodel.NoteChunk { return nil },
		LiveNoteChunksFunc:   func() []appmodel.NoteChunk { return nil },
		SearchLatestNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return searchHits, nil
		},
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return searchHits, nil
		},
		CanReadNoteFunc: func(_ context.Context, n *appmodel.NoteView) (bool, error) {
			return n.Path != "private.md" && n.Path != "_mcp_private.md", nil
		},
		NoteURLFunc:    func(n *appmodel.NoteView) string { return "https://golden.test" + n.Permalink },
		PublicURLFunc:  func() string { return "https://golden.test" },
		LoggerFunc:     func() logger.Logger { return &logger.DummyLogger{} },
		FeaturesFunc:   func() features.Features { return features.Features{} },
		SiteConfigFunc: func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		MCPMetricsFunc: func() *metrics.MCPMetrics { return nil },

		FederationMaxDepthFunc:         func() int { return 3 },
		FederatedFanoutConcurrencyFunc: func() int { return 4 },
		FederatedFanoutLimitFunc:       func() int { return 8 },
		FederatedFanoutTimeoutFunc:     func() time.Duration { return 2 * time.Second },
		FederatedGraphQLEnabledFunc:    func() bool { return false },
		FederationClientFunc: func(context.Context, string) (appmodel.Federation, error) {
			return goldenFedClient{}, nil
		},
		CachedFederatedInstructionsFunc: func(string) (appmodel.FederationResult, bool) {
			return appmodel.FederationResult{}, false
		},
		StoreFederatedInstructionsFunc: func(string, appmodel.FederationResult) {},

		ResolveAPIKeyFunc: func(_ context.Context, value, _ string) (*db.ApiKey, error) {
			switch value {
			case "admin-key":
				return &db.ApiKey{CreatedBy: 42, EnableMcpAdminTools: ptr.To(true)}, nil
			case "plain-key":
				return &db.ApiKey{CreatedBy: 43}, nil
			default:
				return nil, fmt.Errorf("api key not found")
			}
		},
		GraphQLRequestFunc: func(context.Context, string, map[string]any) ([]byte, error) {
			return []byte(`{"data":{"admin":{"__typename":"AdminQuery"}}}`), nil
		},
		GraphQLRequestScopedFunc: func(context.Context, string, map[string]any, []string) ([]byte, error) {
			return []byte(`{"data":{}}`), nil
		},
	}
}

// sortedNames keeps the case list unique and stable so a duplicated name can
// never silently overwrite another case's golden section.
func TestGoldenCaseNamesUnique(t *testing.T) {
	names := make([]string, 0, len(goldenCases()))
	for _, c := range goldenCases() {
		names = append(names, c.name)
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		require.NotEqual(t, sorted[i-1], sorted[i], "duplicate golden case name")
	}
}
