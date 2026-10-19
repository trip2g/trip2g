package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"trip2g/internal/appconfig"
	"trip2g/internal/assetindex"
	"trip2g/internal/case/mcp"
	"trip2g/internal/dataencryption"
	"trip2g/internal/datasize"
	"trip2g/internal/db"
	"trip2g/internal/defaulttemplate"
	"trip2g/internal/localstorage"
	"trip2g/internal/mdloader"
	"trip2g/internal/metrics"
	"trip2g/internal/model"
	"trip2g/internal/movebus"
	"trip2g/internal/notebus"
	"trip2g/internal/noteloader"
	"trip2g/internal/pagecache"
	"trip2g/internal/personaltoken"
	"trip2g/internal/purchasetoken"
	"trip2g/internal/redirectmanager"
	"trip2g/internal/tgbots"
	"trip2g/internal/userbans"
	"trip2g/internal/usertoken"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const (
	fedTestKID    = "consumer-at-peer"
	fedTestSecret = "0123456789abcdef0123456789abcdef"
)

// fedTestZipContent is larger than the asset client's 64 KiB buffer, so the relay
// takes the streaming path rather than a buffered body.
func fedTestZipContent() string {
	return "PK\x03\x04" + strings.Repeat("the attachment the shared note links to ", 4096)
}

// store is one running instance, its handler reachable in process.
type store struct {
	app     *app
	handler fasthttp.RequestHandler
	adminID int64
}

func newStore(t *testing.T, publicURL string) *store {
	t.Helper()
	ctx := context.Background()

	err := defaulttemplate.Init()
	require.NoError(t, err)

	a := newSingleLoaderTestApp(t)
	a.config = &appconfig.Config{
		PublicURL:                 publicURL,
		UserToken:                 usertoken.Config{CookieName: "trip2g_token", Secret: publicURL + "-secret", ExpiresIn: time.Hour, Insecure: true},
		MCPFederationMaxDepth:     3,
		MCPFederationTimeout:      2 * time.Second,
		FederatedFanoutTimeout:    5 * time.Second,
		MCPFederationAllowPrivate: true,
		MCPFederationAssetMaxSize: datasize.Megabyte,
		MCPFederationAssetTimeout: 5 * time.Second,
	}
	encryption, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	require.NoError(t, err)
	a.Manager = encryption
	a.appState.UserBans = userbans.New(a.Queries)
	a.pageCache = pagecache.New()
	a.tokenManager = usertoken.NewManager(a.config.UserToken)
	a.purchaseTokenManager = purchasetoken.NewManager(a.config.PurchaseToken)
	a.mcpMetrics = metrics.NewMCPMetrics(prometheus.NewRegistry())
	a.TgBots = &tgbots.TgBots{}
	a.moveBus = movebus.New(a.log)
	a.noteBus = notebus.New(a.log)
	a.setTokenValidator()
	a.personalTokenResolver = personaltoken.NewResolver(a)
	a.fedHTTPClient = &fasthttp.Client{MaxResponseBodySize: 1 << 20}
	a.fedAssetHTTPClient = newFederatedAssetHTTPClient(true)

	storage, err := localstorage.New(localstorage.Config{Dir: t.TempDir()})
	require.NoError(t, err)
	a.Storage = storage

	user, err := a.WriteQueries.InsertUserWithEmail(ctx, db.InsertUserWithEmailParams{Email: "admin@example.com", CreatedVia: "test"})
	require.NoError(t, err)
	_, err = a.WriteQueries.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
	require.NoError(t, err)

	return &store{app: a, adminID: user.ID}
}

func (s *store) load(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	a := s.app

	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), mdloader.Config{})
	a.liveNoteLoader = a.latestNoteLoader
	err := a.latestNoteLoader.Load(ctx, noteloader.LoadOptions{})
	require.NoError(t, err)

	a.AssetIndex = assetindex.New(a)
	a.setupAssets()
	a.redirectManager, err = redirectmanager.New(ctx, a)
	require.NoError(t, err)
	// Building a handler registers process-wide metrics, and this process builds
	// several.
	registerer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry() //nolint:reassign // restored below
	defer func() {
		prometheus.DefaultRegisterer = registerer //nolint:reassign // the original registry
	}()
	s.handler = a.requestHandler()
}

func (s *store) attach(t *testing.T, versionID int64, path, content string) {
	t.Helper()
	ctx := context.Background()

	assetParams := db.InsertNoteAssetParams{
		AbsolutePath: path,
		FileName:     path[strings.LastIndex(path, "/")+1:],
		Sha256Hash:   sha256Hex(content),
		Size:         int64(len(content)),
	}
	asset, err := s.app.WriteQueries.InsertNoteAsset(ctx, assetParams)
	require.NoError(t, err)

	err = s.app.Storage.PutAssetObject(ctx, strings.NewReader(content), asset)
	require.NoError(t, err)

	linkParams := db.UpsertNoteVersionAssetParams{AssetID: asset.ID, VersionID: versionID, Path: path}
	err = s.app.WriteQueries.UpsertNoteVersionAsset(ctx, linkParams)
	require.NoError(t, err)
}

func (s *store) pairing(t *testing.T, kbURL *string, subgraph string) {
	t.Helper()
	ctx := context.Background()

	crypt, err := s.app.EncryptData([]byte(fedTestSecret))
	require.NoError(t, err)

	secretParams := db.InsertFederationSecretParams{Kid: fedTestKID, SecretCrypt: crypt, KbUrl: kbURL, CreatedBy: s.adminID}
	_, err = s.app.WriteQueries.InsertFederationSecret(ctx, secretParams)
	require.NoError(t, err)

	if subgraph == "" {
		return
	}
	sg, err := s.app.Queries.SubgraphByName(ctx, subgraph)
	require.NoError(t, err)
	grantParams := db.InsertFederationSecretSubgraphParams{Kid: fedTestKID, SubgraphID: sg.ID, CreatedBy: s.adminID}
	err = s.app.WriteQueries.InsertFederationSecretSubgraph(ctx, grantParams)
	require.NoError(t, err)
}

func (s *store) get(t *testing.T, uri string) (int, string, *fasthttp.ResponseHeader) {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodGet)
	fctx.Request.SetRequestURI(uri)
	fctx.Request.Header.SetHost("consumer.test")

	s.handler(fctx)

	return fctx.Response.StatusCode(), string(fctx.Response.Body()), &fctx.Response.Header
}

func (s *store) tool(t *testing.T, name, args string) mcp.Response {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodPost)
	fctx.Request.SetRequestURI("/_system/mcp")
	fctx.Request.Header.SetHost("consumer.test")
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.Header.Set("Accept", "application/json, text/event-stream")
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `}}`
	fctx.Request.SetBodyString(body)

	s.handler(fctx)
	require.Equal(t, http.StatusOK, fctx.Response.StatusCode(), string(fctx.Response.Body()))

	var resp struct {
		Error  *mcp.Error `json:"error"`
		Result struct {
			Content           []mcp.Content   `json:"content"`
			StructuredContent json.RawMessage `json:"structuredContent"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(fctx.Response.Body(), &resp))

	return mcp.Response{Error: resp.Error, Result: mcp.CallToolResult{
		Content:           resp.Result.Content,
		StructuredContent: resp.Result.StructuredContent,
	}}
}

// serve puts a store on a real loopback socket, so the consumer reaches it the
// way it reaches any peer: over HTTP, through its own federation clients.
func serve(t *testing.T, s *store) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		s.handler(ctx)
	}}
	go func() {
		_ = server.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	return "http://" + ln.Addr().String()
}

// TestFederatedAssetAcrossTwoStores runs the whole loop between two instances:
// the consumer reads a peer note, sees its assets, asks for one and downloads it
// from its own URL while the bytes come from the peer.
func TestFederatedAssetAcrossTwoStores(t *testing.T) {
	ctx := context.Background()
	fedTestZip := fedTestZipContent()

	peer := newStore(t, "http://peer.invalid")
	for _, name := range []string{"shared", "other"} {
		require.NoError(t, peer.app.WriteQueries.InsertSubgraph(ctx, name))
	}
	reel := insertLatestNote(t, peer.app, "skills/reel.md",
		"---\nsubgraph: shared\n---\nRead the reel.\n\n[[files/attach.zip]]\n")
	peer.attach(t, reel, "files/attach.zip", fedTestZip)
	plan := insertLatestNote(t, peer.app, "private/plan.md",
		"---\nsubgraph: other\n---\nThe plan.\n\n[[plan.pdf]]\n")
	peer.attach(t, plan, "plan.pdf", "%PDF private")
	peer.pairing(t, nil, "shared")
	peer.load(t)
	peerURL := serve(t, peer)

	consumer := newStore(t, "https://consumer.test")
	kbURL := peerURL + "/_system/mcp"
	insertLatestNote(t, consumer.app, "peer.md",
		"---\nfree: true\nmcp_federation_kb_url: "+kbURL+"\nmcp_federation_kb_id: peer\n---\nThe peer base.\n")
	consumer.pairing(t, &kbURL, "")
	consumer.load(t)

	read := consumer.tool(t, "federated_note_html", `{"kb_id":"peer","path":"skills/reel.md"}`)
	require.Nil(t, read.Error)
	readResult := read.Result.(mcp.CallToolResult)
	var notePayload mcp.NoteHTMLPayload
	require.NoError(t, json.Unmarshal(readResult.StructuredContent.(json.RawMessage), &notePayload))
	wantInfo := model.NoteAssetInfo{Name: "files/attach.zip", Size: int64(len(fedTestZip)), Sha256: sha256Hex(fedTestZip)}
	require.Equal(t, []model.NoteAssetInfo{wantInfo}, notePayload.Assets)
	require.Contains(t, readResult.Content[0].Text, "- files/attach.zip · ")

	got := consumer.tool(t, "federated_asset", `{"kb_id":"peer","note_path":"skills/reel.md","name":"files/attach.zip"}`)
	require.Nil(t, got.Error)
	var assetPayload mcp.FederatedAssetPayload
	require.NoError(t, json.Unmarshal(got.Result.(mcp.CallToolResult).StructuredContent.(json.RawMessage), &assetPayload))
	require.Equal(t, wantInfo.Name, assetPayload.Name)
	require.Equal(t, wantInfo.Size, assetPayload.Size)
	require.Equal(t, wantInfo.Sha256, assetPayload.Sha256)
	require.True(t, strings.HasPrefix(assetPayload.URL, "https://consumer.test/_system/mcp/federated_asset?link="), assetPayload.URL)
	require.WithinDuration(t, time.Now().Add(federatedAssetLinkTTL), assetPayload.ExpiresAt, 5*time.Second)

	link, err := url.Parse(assetPayload.URL)
	require.NoError(t, err)
	status, body, header := consumer.get(t, link.RequestURI())
	require.Equal(t, http.StatusOK, status, body)
	sum := sha256.Sum256([]byte(body))
	require.Equal(t, assetPayload.Sha256, hex.EncodeToString(sum[:]))
	require.Equal(t, `attachment; filename=attach.zip`, string(header.Peek("Content-Disposition")))
	require.Equal(t, "no-store", string(header.Peek("Cache-Control")))

	status, _, _ = consumer.get(t, link.RequestURI())
	require.Equal(t, http.StatusGone, status, "a link downloads once")

	refused := consumer.tool(t, "federated_asset", `{"kb_id":"peer","note_path":"private/plan.md","name":"plan.pdf"}`)
	require.NotNil(t, refused.Error, "a note outside the pairing's grant gives no assets")
	require.Contains(t, refused.Error.Message, `no asset "plan.pdf"`)

	status, _, _ = consumer.get(t, link.Path+"?link="+url.QueryEscape(link.Query().Get("link")+"x"))
	require.Equal(t, http.StatusNotFound, status, "a tampered link reaches nothing")
}

func TestFederatedAssetDownloadNeedsAPublicURL(t *testing.T) {
	a := &app{appState: &appState{config: &appconfig.Config{
		UserToken: usertoken.Config{Secret: "secret"},
	}}}
	params := model.FederatedAssetParams{NotePath: "skills/reel.md", Name: "files/attach.zip"}

	_, err := a.FederatedAssetDownload("peer", params)
	require.EqualError(t, err, "federated_asset needs --public-url")
}
