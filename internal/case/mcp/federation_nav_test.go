package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// The recorded walks that motivate these tests: an agent driving the public
// hub through a text-only client guessed kb_ids like trip2g/markavrelii and
// en/hub/markavrelii because the correct id appeared in no text the server
// returned, and the not-configured error told it to send the same id again.

func pointerNote() *appmodel.NoteView {
	return &appmodel.NoteView{
		Path:               "en/hub/markavrelii.md",
		PathID:             1063,
		Title:              "Marcus Aurelius — Meditations",
		Permalink:          "/en/hub/markavrelii",
		HTML:               "<h1>Marcus Aurelius — Meditations</h1><p>Our own knowledge base.</p>",
		MCPFederationKBURL: "https://markavrelii.2pub.me/_system/mcp",
		MCPFederationKBID:  "markavrelii",
	}
}

func callTool(t *testing.T, env *EnvMock, name, arguments string) mcp.Response {
	t.Helper()
	paramsJSON, err := json.Marshal(mcp.CallToolParams{Name: name, Arguments: json.RawMessage(arguments)})
	require.NoError(t, err)
	return callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})
}

func TestSearchTextNamesFederationPointer(t *testing.T) {
	note := pointerNote()
	env := &EnvMock{
		FeaturesFunc:               func() features.Features { return features.Features{} },
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		SiteConfigFunc:             func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		SearchLiveNotesFunc: func(string) ([]appmodel.SearchResult, error) {
			return []appmodel.SearchResult{{
				NoteView:           note,
				URL:                note.Permalink,
				Score:              0.7,
				HighlightedContent: []string{"Our own knowledge base."},
			}}, nil
		},
		LiveNoteChunksFunc: func() []appmodel.NoteChunk { return nil },
		PublicURLFunc:      func() string { return "https://hub.local" },
		NoteURLFunc:        func(n *appmodel.NoteView) string { return "https://hub.local" + n.Permalink },
		LoggerFunc:         func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc:    func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
	}

	resp := callTool(t, env, "search", `{"query":"marcus"}`)

	require.Nil(t, resp.Error)
	text := resp.Result.(mcp.CallToolResult).Content[0].Text
	require.Contains(t, text, "kind: federation_kb")
	require.Contains(t, text, `kb_id: markavrelii → federated_search(kb_id="markavrelii")`)
}

func TestSimilarTextNamesFederationPointer(t *testing.T) {
	source := &appmodel.NoteView{Path: "en/stoics.md", PathID: 1, VersionID: 1, Permalink: "/en/stoics", Embedding: []float32{1, 0}}
	pointer := pointerNote()
	pointer.VersionID = 2
	pointer.Embedding = []float32{0.9, 0.1}
	nvs := appmodel.NewNoteViews()
	nvs.RegisterNote(source)
	nvs.RegisterNote(pointer)
	nvs.List = []*appmodel.NoteView{source, pointer}
	env := &EnvMock{
		FeaturesFunc: func() features.Features {
			return features.Features{VectorSearch: features.VectorSearchConfig{Enabled: true}}
		},
		FederatedFanoutTimeoutFunc: func() time.Duration { return 2 * time.Second },
		MCPMetricsFunc:             func() *metrics.MCPMetrics { return nil },
		SiteConfigFunc:             func(context.Context) appmodel.SiteConfig { return appmodel.SiteConfig{} },
		LatestNoteViewsFunc:        func() *appmodel.NoteViews { return nvs },
		LatestNoteChunksFunc:       func() []appmodel.NoteChunk { return nil },
		PublicURLFunc:              func() string { return "https://hub.local" },
		NoteURLFunc:                func(n *appmodel.NoteView) string { return "https://hub.local" + n.Permalink },
		LoggerFunc:                 func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc:            func(context.Context, *appmodel.NoteView) (bool, error) { return true, nil },
	}

	resp := callTool(t, env, "similar", `{"path":"en/stoics.md"}`)

	require.Nil(t, resp.Error)
	text := resp.Result.(mcp.CallToolResult).Content[0].Text
	require.Contains(t, text, "Marcus Aurelius")
	require.Contains(t, text, `kb_id: markavrelii → federated_search(kb_id="markavrelii")`)
}

// A pointer seen through a hop must print the composed id: the philosophers
// hub says "montaigne", the caller has to send "philosophers/montaigne".
func TestFederatedSearchComposesPointerHintThroughHop(t *testing.T) {
	fed := &federationMock{
		searchFunc: func(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "Found 1 notes:\n\n" +
					"1. Montaigne — Essays\n   en/hub/montaigne.md\n   https://philosophers.2pub.me/en/hub/montaigne\n" +
					`   kind: federation_kb · kb_id: montaigne → federated_search(kb_id="montaigne")` + "\n\n"}},
				StructuredContent: json.RawMessage(`{"results":[{"title":"Montaigne — Essays","kind":"federation_kb","kb_id":"montaigne",` +
					`"federation":{"kb_id":"montaigne","kb_url":"https://montaigne.2pub.me/_system/mcp",` +
					`"agent_instruction":"This is a knowledge base pointer. To search inside it, call federated_search with kb_id=\"montaigne\". To open notes from it, call federated_note_html(path=..., kb_id=\"montaigne\")."}}]}`),
			}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"essays","kb_id":"philosophers"}`)

	text := result.Content[0].Text
	require.Contains(t, text, `kb_id: philosophers/montaigne → federated_search(kb_id="philosophers/montaigne")`)
	require.NotContains(t, text, `kb_id: montaigne →`)
	payload := fedSingleSearchPayload(t, result)
	require.Equal(t, "philosophers/montaigne", payload.Results[0].KBID)
	require.Contains(t, payload.Results[0].Federation.AgentInstruction, `kb_id="philosophers/montaigne"`)
	require.NotContains(t, payload.Results[0].Federation.AgentInstruction, `kb_id="montaigne"`)
}

func TestNoteHTMLNamesFederationPointer(t *testing.T) {
	note := pointerNote()
	env := noteHTMLEnv(note)
	env.LatestNoteChunksFunc = func() []appmodel.NoteChunk {
		return []appmodel.NoteChunk{{NotePath: note.Path, ChunkIndex: 0, Content: "Our own knowledge base."}}
	}

	tests := []struct {
		name string
		args string
	}{
		{name: "whole note", args: `{"path":"en/hub/markavrelii.md"}`},
		{name: "focused chunk", args: `{"match_id":"p1063:c0"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := callTool(t, env, "note_html", tt.args)

			require.Nil(t, resp.Error)
			text := resp.Result.(mcp.CallToolResult).Content[0].Text
			require.NotEmpty(t, text)
			require.Contains(t, text, `federation pointer · kb_id: markavrelii → federated_search(kb_id="markavrelii")`)
			require.Contains(t, text, "Our own knowledge base.")
		})
	}
}

func TestFederatedNoteHTMLComposesPointerLineThroughHop(t *testing.T) {
	fed := &federationMock{
		noteHTMLFunc: func(context.Context, appmodel.MCPNoteHTMLParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{Content: []appmodel.FederationContent{{
				Type: "text",
				Text: "federation pointer · kb_id: montaigne → federated_search(kb_id=\"montaigne\")\n\n<h1>Montaigne</h1>",
			}}}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	resp := callTool(t, env, "federated_note_html", `{"kb_id":"philosophers","path":"en/hub/montaigne.md"}`)

	require.Nil(t, resp.Error)
	text := resp.Result.(mcp.CallToolResult).Content[0].Text
	require.Contains(t, text, `federation pointer · kb_id: philosophers/montaigne → federated_search(kb_id="philosophers/montaigne")`)
	require.Contains(t, text, "<h1>Montaigne</h1>")
}

// In the third recorded walk the model sent the previous call's match_id
// together with a new toc_path three times and got the same chunk back every
// time: match_id was resolved first. An explicit toc_path is navigation and
// has to win.
func TestNoteHTMLTocPathWinsOverMatchID(t *testing.T) {
	note := &appmodel.NoteView{
		Path:      "en/user/memory.md",
		PathID:    1316,
		Permalink: "/en/user/memory",
		HTML: `<div data-header="Step 3. Register"><h2>Step 3. Register</h2><p>claude mcp add trip2g-memory</p></div>` +
			`<div data-header="Step 4. Verify"><h2>Step 4. Verify</h2><p>Run /mcp</p></div>`,
		Headings: appmodel.NoteViewHeadings{
			{Text: "Step 3. Register", Level: 2, ID: "step-3"},
			{Text: "Step 4. Verify", Level: 2, ID: "step-4"},
		},
	}
	env := noteHTMLEnv(note)
	env.LatestNoteChunksFunc = func() []appmodel.NoteChunk {
		return []appmodel.NoteChunk{{NotePath: note.Path, ChunkIndex: 12, Content: "Step 4. Verify\n\nRun /mcp"}}
	}

	tests := []struct {
		name        string
		args        string
		wantText    string
		wantMissing string
		wantErr     string
	}{
		{
			name:        "toc_path wins over a valid match_id",
			args:        `{"path":"en/user/memory.md","match_id":"p1316:c12","toc_path":["Step 3. Register"]}`,
			wantText:    "claude mcp add trip2g-memory",
			wantMissing: "Run /mcp",
		},
		{
			name:     "match_id alone reads the focused chunk",
			args:     `{"path":"en/user/memory.md","match_id":"p1316:c12"}`,
			wantText: "Run /mcp",
		},
		{
			name:     "empty toc_path leaves match_id in charge",
			args:     `{"path":"en/user/memory.md","match_id":"p1316:c12","toc_path":[]}`,
			wantText: "Run /mcp",
		},
		{
			name:    "a toc_path miss fails loud even with a valid match_id",
			args:    `{"path":"en/user/memory.md","match_id":"p1316:c12","toc_path":["Step 5. Teach"]}`,
			wantErr: "section not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := callTool(t, env, "note_html", tt.args)

			if tt.wantErr != "" {
				require.NotNil(t, resp.Error)
				require.Contains(t, resp.Error.Message, tt.wantErr)
				return
			}
			require.Nil(t, resp.Error)
			text := resp.Result.(mcp.CallToolResult).Content[0].Text
			require.Contains(t, text, tt.wantText)
			if tt.wantMissing != "" {
				require.NotContains(t, text, tt.wantMissing)
			}
		})
	}
}

// In the first walk the model fed a confucius match_id to the local note_html
// and got a bare "Note not found". The error has to say where such ids resolve.
func TestNoteHTMLForeignMatchIDHintsFederatedNoteHTML(t *testing.T) {
	env := noteHTMLEnv(&appmodel.NoteView{Path: "en/local.md", PathID: 1, HTML: "<p>local</p>"})

	resp := callTool(t, env, "note_html", `{"path":"principles/ritual-i-forma.md","match_id":"p34:c1"}`)

	require.NotNil(t, resp.Error)
	require.Equal(t, mcp.ErrCodeInvalidParams, resp.Error.Code)
	require.Contains(t, resp.Error.Message, "Note not found")
	require.Contains(t, resp.Error.Message, "federated_note_html")
	require.Contains(t, resp.Error.Message, `match_id="p34:c1"`)
}

func TestFederatedSearchNotConfiguredNamesUnknownSegmentAndConnectedBases(t *testing.T) {
	env := fanoutEnv(2, 10, 5, time.Second, func(string) fanoutSearchFn {
		return func(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			t.Fatal("no peer must be called for an unknown kb_id")
			return appmodel.FederationResult{}, nil
		}
	})
	env.FederationMaxDepthFunc = func() int { return 3 }

	tests := []struct {
		name        string
		kbID        string
		want        []string
		wantMissing []string
	}{
		{
			name: "guessed hub prefix on a base this hub connects directly",
			kbID: "trip2g/kb2",
			want: []string{
				`no connected base on this hub is named "trip2g"`,
				"Connected bases: kb1, kb2",
				`"kb2" is a connected base — address it as kb_id="kb2"`,
			},
			wantMissing: []string{`address this base as "trip2g/kb2"`, "<hub>/"},
		},
		{
			name: "flat id that is no local peer keeps the nesting hint",
			kbID: "ghost",
			want: []string{
				`no connected base on this hub is named "ghost"`,
				"Connected bases: kb1, kb2",
				"<hub>/ghost",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := callFederatedSearch(t, env, `{"query":"x","kb_id":"`+tt.kbID+`"}`)

			text := result.Content[0].Text
			require.Contains(t, text, `Federation is not configured for kb_id "`+tt.kbID+`"`)
			for _, w := range tt.want {
				require.Contains(t, text, w)
			}
			for _, w := range tt.wantMissing {
				require.NotContains(t, text, w)
			}
			status := decodePayload[mcp.FederationStatusPayload](t, result)
			require.Equal(t, "federation_not_configured", status.Status)
			require.Equal(t, tt.kbID, status.KBID)
			require.Equal(t, []string{"kb1", "kb2"}, status.ConnectedKBIDs)
		})
	}
}

// A miss reported by a peer is rewritten into the caller's frame: the message
// names the hub that has no such base and lists that hub's bases as the caller
// has to address them.
func TestFederatedSearchNotConfiguredThroughHopListsHubBases(t *testing.T) {
	fed := &federationMock{
		federatedSearchFunc: func(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content: []appmodel.FederationContent{{Type: "text", Text: "not configured"}},
				StructuredContent: json.RawMessage(`{"status":"federation_not_configured","kb_id":"marcus-aurelius",` +
					`"connected_kb_ids":["epictetus","montaigne"]}`),
			}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"x","kb_id":"philosophers/marcus-aurelius"}`)

	text := result.Content[0].Text
	require.Contains(t, text, `Federation is not configured for kb_id "philosophers/marcus-aurelius"`)
	require.Contains(t, text, `hub "philosophers" has no base "marcus-aurelius"`)
	require.Contains(t, text, `Bases connected under "philosophers": philosophers/epictetus, philosophers/montaigne`)
	require.NotContains(t, text, `address this base as`)
	status := decodePayload[mcp.FederationStatusPayload](t, result)
	require.Equal(t, "philosophers/marcus-aurelius", status.KBID)
	require.Equal(t, "philosophers", status.Hub)
	require.Equal(t, []string{"philosophers/epictetus", "philosophers/montaigne"}, status.ConnectedKBIDs)
}

// A peer that predates the connected list still gets a hint that works: search
// the hub for the base, its pointer card prints the kb_id in the caller's frame.
func TestFederatedSearchNotConfiguredThroughHopWithoutListPointsAtHubSearch(t *testing.T) {
	fed := &federationMock{
		federatedSearchFunc: func(context.Context, appmodel.MCPSearchParams) (appmodel.FederationResult, error) {
			return appmodel.FederationResult{
				Content:           []appmodel.FederationContent{{Type: "text", Text: "not configured"}},
				StructuredContent: json.RawMessage(`{"status":"federation_not_configured","kb_id":"ghost"}`),
			}, nil
		},
	}
	env := singlePeerEnv("philosophers", fed)

	result := callFederatedSearch(t, env, `{"query":"x","kb_id":"philosophers/ghost"}`)

	text := result.Content[0].Text
	require.Contains(t, text, `hub "philosophers" has no base "ghost"`)
	require.Contains(t, text, `federated_search(kb_id="philosophers", query="ghost")`)
}
