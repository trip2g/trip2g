package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
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

// rpc builds a JSON-RPC request body; call builds a tools/call one. They keep
// the case table readable at a glance — the interesting part of each case is
// the arguments, not the envelope around them.
func rpc(id int, method, params string) string {
	if params == "" {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q}`, id, method)
	}
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q,"params":%s}`, id, method, params)
}

func call(id int, tool, args string) string {
	return rpc(id, "tools/call", fmt.Sprintf(`{"name":%q,"arguments":%s}`, tool, args))
}

func goldenCases() []goldenCase {
	const clientInit = `{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"golden","version":"1"}}`

	return []goldenCase{
		{name: "initialize", body: rpc(1, "initialize", clientInit)},
		{name: "initialize_no_params", body: rpc(1, "initialize", "")},
		{name: "initialize_method_override", body: rpc(1, "initialize", ""), query: "method=guide"},
		{name: "initialize_method_override_unknown", body: rpc(1, "initialize", ""), query: "method=nope"},
		{name: "initialize_method_override_private", body: rpc(1, "initialize", ""), query: "method=private-tool"},
		{name: "notifications_initialized", body: `{"jsonrpc":"2.0","id":null,"method":"notifications/initialized"}`},

		{name: "tools_list_anonymous", body: rpc(2, "tools/list", "{}")},
		{name: "tools_list_admin_api_key", body: rpc(2, "tools/list", "{}"), headers: adminKey},
		{name: "tools_list_plain_api_key", body: rpc(2, "tools/list", "{}"), headers: plainKey},

		{name: "call_search", body: call(3, "search", `{"query":"alpha"}`)},
		{name: "call_search_limits", body: call(3, "search", `{"query":"alpha","limit":99,"detail_limit":1}`)},
		{name: "call_search_missing_query", body: call(3, "search", `{}`)},
		{name: "call_search_wrong_arg_type", body: call(3, "search", `{"query":42}`)},

		{name: "call_note_html_by_path", body: call(4, "note_html", `{"path":"doc.md"}`)},
		{name: "call_note_html_by_pid_number", body: call(4, "note_html", `{"pid":10}`)},
		{name: "call_note_html_by_pid_string", body: call(4, "note_html", `{"pid":"10"}`)},
		{name: "call_note_html_chunk_ref_in_pid", body: call(4, "note_html", `{"pid":"p10:c2"}`)},
		{name: "call_note_html_toc_path", body: call(4, "note_html", `{"path":"doc.md","toc_path":["Beta"]}`)},
		{name: "call_note_html_toc_path_miss", body: call(4, "note_html", `{"path":"doc.md","toc_path":["Nope"]}`)},
		{name: "call_note_html_no_selector", body: call(4, "note_html", `{}`)},
		{name: "call_note_html_not_found", body: call(4, "note_html", `{"path":"missing.md"}`)},
		{name: "call_note_html_private", body: call(4, "note_html", `{"path":"private.md"}`)},

		{name: "call_expand_root", body: call(5, "expand", `{"path":"doc.md"}`)},
		{name: "call_expand_section", body: call(5, "expand", `{"path":"doc.md","toc_path":["Beta"]}`)},
		{name: "call_expand_no_selector", body: call(5, "expand", `{}`)},
		{name: "call_expand_leading_h1", body: call(5, "expand", `{"path":"aphorisms.md"}`)},
		{name: "call_note_html_h1_shortened_toc_path", body: call(5, "note_html", `{"path":"aphorisms.md","toc_path":["1"]}`)},

		{name: "call_federated_search", body: call(6, "federated_search", `{"query":"alpha"}`)},
		{name: "call_federated_search_by_kb", body: call(6, "federated_search", `{"query":"alpha","kb_id":"peer"}`)},
		{name: "call_federated_search_unknown_kb", body: call(6, "federated_search", `{"query":"alpha","kb_id":"ghost"}`)},
		{name: "call_federated_instructions", body: call(6, "federated_instructions", `{"kb_id":"peer"}`)},

		{name: "call_graphql_request_unauthorized", body: call(7, "graphql_request", gqlArgs)},
		{name: "call_graphql_introspection_unauthorized", body: call(7, "graphql_introspection", `{"pattern":"Note"}`)},
		{name: "call_graphql_request_admin", body: call(7, "graphql_request", gqlArgs), headers: adminKey},

		{name: "call_dynamic_tool", body: call(8, "code-review", `{}`)},
		{name: "call_dynamic_tool_private", body: call(8, "private-tool", `{}`)},
		{name: "call_unknown_tool", body: call(8, "does-not-exist", `{}`)},
		{name: "call_missing_params", body: rpc(8, "tools/call", "")},

		{name: "unknown_method", body: rpc(9, "resources/list", "{}")},
		{name: "unimplemented_method", body: rpc(9, "totally/unknown", "{}")},
		{name: "bad_jsonrpc_version", body: `{"jsonrpc":"1.0","id":9,"method":"tools/list","params":{}}`},
		{name: "parse_error", body: `{not json`},
		{name: "string_id", body: `{"jsonrpc":"2.0","id":"abc","method":"tools/list","params":{}}`},

		{name: "federation_depth_ok", body: rpc(10, "tools/list", "{}"), headers: depth("2")},
		{name: "federation_depth_exceeded", body: rpc(10, "tools/list", "{}"), headers: depth("9")},
		{name: "federation_depth_malformed", body: rpc(10, "tools/list", "{}"), headers: depth("abc")},

		{name: "bad_api_key", body: rpc(11, "tools/list", "{}"), headers: apiKey("nope")},
		{name: "malformed_bearer", body: rpc(11, "tools/list", "{}"), headers: map[string]string{"Authorization": "Basic zzz"}},

		// Streamable HTTP requires clients to accept both media types; a client
		// that omits Accept is rejected by the transport before dispatch.
		{name: "missing_accept_header", body: rpc(12, "tools/list", "{}"), noAccept: true},
	}
}

const gqlArgs = `{"query":"{admin{__typename}}"}`

func apiKey(value string) map[string]string { return map[string]string{"X-API-Key": value} }

func depth(value string) map[string]string {
	return map[string]string{"X-MCP-Federation-Depth": value}
}

var (
	adminKey = apiKey("admin-key") //nolint:gochecknoglobals // test fixture
	plainKey = apiKey("plain-key") //nolint:gochecknoglobals // test fixture
)

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

	req := newMCPRequest(fasthttpCtx, goldenEnv())
	defer appreq.Release(req)

	_, err := (&mcp.Endpoint{}).Handle(req)
	require.NoError(t, err)

	return fasthttpCtx.Response.StatusCode(),
		string(fasthttpCtx.Response.Header.ContentType()),
		normalizeGolden(fasthttpCtx.Response.Body())
}

// latencyRe blanks the wall-clock latency federation results carry, which is
// the only nondeterministic field in the matrix.
var latencyRe = regexp.MustCompile(`"latency":\s*"[^"]*"`)

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

func (goldenFedClient) Search(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer search hit"), nil
}

func (goldenFedClient) FederatedSearch(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer federated search hit"), nil
}

func (goldenFedClient) Instructions(context.Context) (appmodel.FederationResult, error) {
	return goldenFedResult("peer instructions"), nil
}

func (goldenFedClient) FederatedInstructions(context.Context, appmodel.MCPInstructionsParams) (appmodel.FederationResult, error) {
	return goldenFedResult("peer federated instructions"), nil
}

const goldenPeerPayload = `{"query":"alpha","results":[{"title":"Peer Note","note_id":1,` +
	`"note_path":"peer.md","href":"/peer","url":"https://peer.example/peer",` +
	`"kind":"note","score":1}]}`

func goldenFedResult(text string) appmodel.FederationResult {
	return appmodel.FederationResult{
		Content:           []appmodel.FederationContent{{Type: "text", Text: text}},
		StructuredContent: json.RawMessage(goldenPeerPayload),
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
	// Mirrors the real load pipeline for a note that opens with an H1: HasH1 is
	// set, the H1 heads the Headings list, and its data-header div wraps the
	// rest of the document. Aphoristic section titles pin the short-heading
	// previews at the same time.
	aphorisms := &appmodel.NoteView{
		Path:      "aphorisms.md",
		PathID:    20,
		Title:     "Книга 10",
		Permalink: "/aphorisms",
		HasH1:     true,
		HTML: template.HTML(`<div data-header="Книга 10" data-level="1"><h1 id="kniga">Книга 10</h1>` +
			`<div data-header="1" data-level="2"><h2 id="one">1</h2><p>Душа моя, ужели ты никогда не будешь доброй и простой?</p></div>` +
			`<div data-header="2" data-level="2"><h2 id="two">2</h2><p>Замечай, чего требует твоя природа.</p></div></div>`),
		Content: []byte("# Книга 10\n\n## 1\n\nДуша моя.\n\n## 2\n\nЗамечай.\n"),
		Headings: appmodel.NoteViewHeadings{
			{Text: "Книга 10", Level: 1, ID: "kniga"},
			{Text: "1", Level: 2, ID: "one"},
			{Text: "2", Level: 2, ID: "two"},
		},
	}
	// A tool note without the leading-underscore naming convention: the
	// path-based system-note rule misses it, so it used to surface in search.
	toolNote := &appmodel.NoteView{
		Path:      "instructions.md",
		PathID:    21,
		Title:     "Instructions",
		Permalink: "/instructions",
		MCPMethod: "instructions",
		Content:   []byte("---\nmcp_method: instructions\n---\n\nBase instructions."),
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

	notes := []*appmodel.NoteView{doc, aphorisms, toolNote, private, initNote, guide, codeReview, privateTool, peer}
	views := appmodel.NewNoteViews()
	views.List = notes
	for _, n := range notes {
		views.PathMap[n.Path] = n
		if n.Permalink != "" {
			views.Map[n.Permalink] = n
		}
	}
	views.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(peer)}

	searchHits := []appmodel.SearchResult{
		{
			NoteView:           doc,
			URL:                doc.Permalink,
			Score:              0.9,
			HighlightedContent: []string{"alpha body"},
		},
		{
			NoteView:           toolNote,
			URL:                toolNote.Permalink,
			Score:              0.4,
			HighlightedContent: []string{"Base instructions."},
		},
	}

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
				return nil, errors.New("api key not found")
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
