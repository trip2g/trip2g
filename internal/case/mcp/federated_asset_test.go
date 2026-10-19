package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"trip2g/internal/case/mcp"
	"trip2g/internal/features"
	"trip2g/internal/logger"
	"trip2g/internal/metrics"
	appmodel "trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

type assetFetcherMock struct {
	describe func(params appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error)
}

func (m *assetFetcherMock) DescribeAsset(_ context.Context, params appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
	return m.describe(params)
}

func (m *assetFetcherMock) FetchAsset(context.Context, appmodel.FederatedAssetParams) (*appmodel.FederatedAsset, error) {
	panic("the tool never fetches the bytes")
}

func federatedAssetEnv(t *testing.T, canReadKB bool, fetcher *assetFetcherMock) *EnvMock {
	t.Helper()

	kbNote := &appmodel.NoteView{
		PathID:             17,
		MCPFederationKBURL: "https://bob.example/_system/mcp",
		MCPFederationKBID:  "bob",
	}
	nvs := appmodel.NewNoteViews()
	nvs.MCPFederationNotes = []*appmodel.MCPFederationNote{appmodel.NewMCPFederationNote(kbNote)}

	return &EnvMock{
		FeaturesFunc:        func() features.Features { return features.Features{} },
		MCPMetricsFunc:      func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		CanReadNoteFunc: func(context.Context, *appmodel.NoteView) (bool, error) {
			return canReadKB, nil
		},
		FederationAssetFetcherFunc: func(_ context.Context, kbID string) (appmodel.FederationAssetFetcher, error) {
			require.Equal(t, "bob", kbID)
			return fetcher, nil
		},
		FederatedAssetDownloadFunc: func(kbID string, params appmodel.FederatedAssetParams) (mcp.FederatedAssetDownload, error) {
			require.Equal(t, "bob", kbID)
			download := mcp.FederatedAssetDownload{
				URL:       "https://hub.example/_system/mcp/federated_asset?link=signed",
				ExpiresAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			}
			return download, nil
		},
	}
}

func callFederatedAsset(t *testing.T, env mcp.Env, args string) mcp.Response {
	t.Helper()

	params := mcp.CallToolParams{Name: "federated_asset", Arguments: json.RawMessage(args)}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	return callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})
}

func TestFederatedAssetDescribesAndLinks(t *testing.T) {
	var asked appmodel.FederatedAssetParams
	fetcher := &assetFetcherMock{describe: func(params appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
		asked = params
		return appmodel.NoteAssetInfo{Name: params.Name, Size: 42, Sha256: "abc123"}, nil
	}}
	env := federatedAssetEnv(t, true, fetcher)

	resp := callFederatedAsset(t, env, `{"kb_id":"bob","note_path":"skills/reel.md","name":"files/attach.zip"}`)
	require.Nil(t, resp.Error)

	require.Equal(t, appmodel.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"}, asked)

	result := resp.Result.(mcp.CallToolResult)
	var payload mcp.FederatedAssetPayload
	require.NoError(t, json.Unmarshal(result.StructuredContent.(json.RawMessage), &payload))
	require.Equal(t, mcp.FederatedAssetPayload{
		Name:      "files/attach.zip",
		Size:      42,
		Sha256:    "abc123",
		URL:       "https://hub.example/_system/mcp/federated_asset?link=signed",
		ExpiresAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}, payload)
	require.Contains(t, result.Content[0].Text,
		"curl -fsS -o 'attach.zip' 'https://hub.example/_system/mcp/federated_asset?link=signed'")
	require.Contains(t, result.Content[0].Text, "sha256 abc123")
}

func TestFederatedAssetRefusals(t *testing.T) {
	notFound := func(appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
		return appmodel.NoteAssetInfo{}, &appmodel.FederationHTTPError{Status: http.StatusNotFound}
	}
	tooLarge := func(appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
		return appmodel.NoteAssetInfo{}, appmodel.ErrFederatedAssetTooLarge
	}
	unreachable := func(appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
		panic("the peer must not be asked")
	}

	cases := []struct {
		name      string
		canReadKB bool
		describe  func(appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error)
		args      string
		code      int
		message   string
	}{
		{
			name: "the peer has no such asset for this pairing", canReadKB: true, describe: notFound,
			args:    `{"kb_id":"bob","note_path":"a.md","name":"x.zip"}`,
			code:    mcp.ErrCodeInvalidParams,
			message: `no asset "x.zip" in note "a.md" of "bob"`,
		},
		{
			name: "over the size limit", canReadKB: true, describe: tooLarge,
			args:    `{"kb_id":"bob","note_path":"a.md","name":"x.zip"}`,
			code:    mcp.ErrCodeInvalidParams,
			message: "larger than federation is allowed to move",
		},
		{
			name: "a nested base", canReadKB: true, describe: unreachable,
			args:    `{"kb_id":"bob/carol","note_path":"a.md","name":"x.zip"}`,
			code:    mcp.ErrCodeInvalidParams,
			message: "directly connected base only",
		},
		{
			name: "a missing name", canReadKB: true, describe: unreachable,
			args:    `{"kb_id":"bob","note_path":"a.md"}`,
			code:    mcp.ErrCodeInvalidParams,
			message: "kb_id, note_path and name are required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := federatedAssetEnv(t, tc.canReadKB, &assetFetcherMock{describe: tc.describe})

			resp := callFederatedAsset(t, env, tc.args)
			require.NotNil(t, resp.Error)
			require.Equal(t, tc.code, resp.Error.Code)
			require.Contains(t, resp.Error.Message, tc.message)
		})
	}
}

func TestFederatedAssetNeedsTheKBNote(t *testing.T) {
	fetcher := &assetFetcherMock{describe: func(appmodel.FederatedAssetParams) (appmodel.NoteAssetInfo, error) {
		panic("a caller who cannot read the KB-note never reaches the peer")
	}}
	env := federatedAssetEnv(t, false, fetcher)

	resp := callFederatedAsset(t, env, `{"kb_id":"bob","note_path":"a.md","name":"x.zip"}`)
	require.Nil(t, resp.Error)

	result := resp.Result.(mcp.CallToolResult)
	require.Contains(t, result.Content[0].Text, "bob")
	require.Empty(t, env.FederatedAssetDownloadCalls())
}

func TestNoteHTMLListsAssetsByTheirNoteName(t *testing.T) {
	note := &appmodel.NoteView{
		Path:   "skills/reel.md",
		PathID: 5,
		HTML:   "<p>body</p>",
		AssetReplaces: map[string]*appmodel.NoteAssetReplace{
			"files/attach.zip": {ID: 1, Hash: "bbb", FileName: "attach.zip", Size: 300},
			"cover.png":        {ID: 2, Hash: "aaa", FileName: "cover.png", Size: 10},
		},
	}
	nvs := appmodel.NewNoteViews()
	nvs.List = []*appmodel.NoteView{note}
	nvs.PathMap[note.Path] = note

	env := &EnvMock{
		FeaturesFunc:        func() features.Features { return features.Features{} },
		MCPMetricsFunc:      func() *metrics.MCPMetrics { return nil },
		LatestNoteViewsFunc: func() *appmodel.NoteViews { return nvs },
		LoggerFunc:          func() logger.Logger { return &logger.DummyLogger{} },
		CanReadNoteFunc: func(context.Context, *appmodel.NoteView) (bool, error) {
			return true, nil
		},
	}

	params := mcp.CallToolParams{Name: "note_html", Arguments: json.RawMessage(`{"path":"skills/reel.md"}`)}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	resp := callMCP(t, env, mcp.Request{JSONRPC: "2.0", Method: "tools/call", Params: paramsJSON, ID: 1})
	require.Nil(t, resp.Error)

	result := resp.Result.(mcp.CallToolResult)
	var payload mcp.NoteHTMLPayload
	require.NoError(t, json.Unmarshal(result.StructuredContent.(json.RawMessage), &payload))
	require.Equal(t, "skills/reel.md", payload.NotePath)
	require.Equal(t, []appmodel.NoteAssetInfo{
		{Name: "cover.png", Size: 10, Sha256: "aaa"},
		{Name: "files/attach.zip", Size: 300, Sha256: "bbb"},
	}, payload.Assets)

	require.Equal(t, "<p>body</p>\n\nAssets (name · bytes · sha256):\n- cover.png · 10 · aaa\n- files/attach.zip · 300 · bbb",
		result.Content[0].Text)
}
