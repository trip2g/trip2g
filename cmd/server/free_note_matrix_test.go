package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"trip2g/internal/appconfig"
	"trip2g/internal/assetindex"
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

	"github.com/kr/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type matrixNote struct {
	key       string
	path      string
	permalink string
	content   string
	hidden    bool
	noAsset   bool
}

func (n matrixNote) marker() string {
	return "zz" + n.key + "zz"
}

func (n matrixNote) title() string {
	return "Title " + n.key + "x"
}

func (n matrixNote) asset() string {
	return n.key + ".png"
}

func matrixNoteSource(key, frontmatter, extra string) string {
	note := matrixNote{key: key}
	return "---\ntitle: " + note.title() + "\n" + frontmatter + "---\n" +
		"matrixword " + note.marker() + "\n\n![[" + note.asset() + "]]\n\n[[hub]]\n" + extra
}

func matrixNotes() []matrixNote {
	return []matrixNote{
		{
			key: "n1", path: "signin-free.md", permalink: "/signin_free",
			content: matrixNoteSource("n1", "free: true\nsubgraph: members\n", ""),
		},
		{
			key: "n2", path: "signin-closed.md", permalink: "/signin_closed",
			content: matrixNoteSource("n2", "subgraph: members\n", ""),
		},
		{
			key: "n3", path: "paid-free.md", permalink: "/paid_free",
			content: matrixNoteSource("n3", "free: true\nsubgraph: paid\n", ""),
		},
		{
			key: "n4", path: "paid-closed.md", permalink: "/paid_closed",
			content: matrixNoteSource("n4", "subgraph: paid\n", ""),
		},
		{
			key: "n5", path: "open-free.md", permalink: "/open_free",
			content: matrixNoteSource("n5", "free: true\n", ""),
		},
		{
			key: "n6", path: "open-closed.md", permalink: "/open_closed",
			content: matrixNoteSource("n6", "", ""),
		},
		{
			key: "n7", path: "hidden-free.md", permalink: "/hidden_free",
			content: matrixNoteSource("n7", "free: true\n", ""),
			hidden:  true,
		},
		{
			key: "n8", path: "free-page.html", permalink: "/free-page.html",
			content: "---\nfree: true\n---\n<p>matrixword zzn8zz</p>\n",
			noAsset: true,
		},
		{
			key: "n9a", path: "paid-closed-embeds-free.md", permalink: "/paid_closed_embeds_free",
			content: matrixNoteSource("n9a", "subgraph: paid\n", "\n![[paid-free]]\n\n[[open-free]]\n"),
		},
		{
			key: "n9b", path: "free-embeds-paid-closed.md", permalink: "/free_embeds_paid_closed",
			content: matrixNoteSource("n9b", "free: true\n", "\n![[paid-closed]]\n\n[[paid-closed]]\n"),
		},
		{
			key: "n9c", path: "signin-free-embeds-signin-closed.md", permalink: "/signin_free_embeds_signin_closed",
			content: matrixNoteSource("n9c", "free: true\nsubgraph: members\n", "\n![[signin-closed]]\n"),
		},
		{
			key: "n10", path: "signin-free-noindex.md", permalink: "/signin_free_noindex",
			content: matrixNoteSource("n10", "free: true\nsubgraph: members\nnoindex: true\n", ""),
		},
	}
}

type matrixReader struct {
	name   string
	userID int64
	role   string
	grants []string
}

type freeNoteMatrix struct {
	app     *app
	handler fasthttp.RequestHandler
	notes   []matrixNote
	readers []matrixReader
	cookies map[string]string
	assets  map[string]string
}

func seedFreeNoteMatrix(t *testing.T) *freeNoteMatrix {
	t.Helper()
	ctx := context.Background()

	err := defaulttemplate.Init()
	require.NoError(t, err)

	a := newSingleLoaderTestApp(t)
	a.config = &appconfig.Config{
		PublicURL: "https://example.com",
		UserToken: usertoken.Config{CookieName: "trip2g_token", Secret: "matrix-secret", ExpiresIn: time.Hour, Insecure: true},
	}
	a.config.Features.VectorSearch.Enabled = true
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

	storage, err := localstorage.New(localstorage.Config{Dir: t.TempDir()})
	require.NoError(t, err)
	a.Storage = storage

	for _, name := range []string{"members", "paid", "elsewhere"} {
		insertErr := a.WriteQueries.InsertSubgraph(ctx, name)
		require.NoError(t, insertErr)
	}
	members, err := a.Queries.SubgraphByName(ctx, "members")
	require.NoError(t, err)
	signinParams := db.UpdateAdminSubgraphParams{ID: members.ID, RequireSignin: true}
	_, err = a.WriteQueries.UpdateAdminSubgraph(ctx, signinParams)
	require.NoError(t, err)

	m := &freeNoteMatrix{
		app:   a,
		notes: matrixNotes(),
		readers: []matrixReader{
			{name: "guest"},
			{name: "member", role: "user"},
			{name: "outsider", role: "user", grants: []string{"elsewhere"}},
			{name: "insider", role: "user", grants: []string{"members", "paid"}},
			{name: "admin", role: "admin"},
		},
		cookies: map[string]string{},
		assets:  map[string]string{},
	}

	m.seedReaders(t)
	m.seedNotes(t)

	a.latestNoteLoader = noteloader.New("latest", makeLatestNoteLoaderWrapper(a), mdloader.Config{})
	a.liveNoteLoader = a.latestNoteLoader
	err = a.latestNoteLoader.Load(ctx, noteloader.LoadOptions{})
	require.NoError(t, err)

	a.AssetIndex = assetindex.New(a)
	a.setupAssets()
	a.redirectManager, err = redirectmanager.New(ctx, a)
	require.NoError(t, err)
	m.handler = a.requestHandler()

	return m
}

func (m *freeNoteMatrix) seedNotes(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	a := m.app

	rssLayout, err := os.ReadFile("../../onboarding-vault/_layouts/rss.html")
	require.NoError(t, err)
	insertLatestNote(t, a, "_layouts/rss.html", string(rssLayout))
	insertLatestNote(t, a, "feed.md", "---\nfree: true\nlayout: rss\ncontent_type: application/rss+xml\nrss_limit: 100\n---\n")
	hubID := insertLatestNote(t, a, "hub.md", "---\nfree: true\nright_sidebar:\n  - backlinks\n---\nhub\n")
	m.embed(t, hubID)

	for _, note := range m.notes {
		versionID := insertLatestNote(t, a, note.path, note.content)
		m.embed(t, versionID)

		if !note.noAsset {
			m.attachAsset(t, versionID, note.asset())
		}

		if note.hidden {
			hideParams := db.HideNotePathParams{HiddenBy: &m.readers[len(m.readers)-1].userID, Value: note.path}
			err = a.WriteQueries.HideNotePath(ctx, hideParams)
			require.NoError(t, err)
		}
	}
}

func (m *freeNoteMatrix) embed(t *testing.T, versionID int64) {
	t.Helper()

	params := db.UpsertNoteVersionEmbeddingParams{
		VersionID:   versionID,
		Embedding:   model.Float32SliceToBytes([]float32{1, 0, 0}),
		ContentHash: []byte("matrix"),
	}
	err := m.app.WriteQueries.UpsertNoteVersionEmbedding(context.Background(), params)
	require.NoError(t, err)
}

func (m *freeNoteMatrix) attachAsset(t *testing.T, versionID int64, fileName string) {
	t.Helper()
	ctx := context.Background()
	content := "png:" + fileName

	assetParams := db.InsertNoteAssetParams{
		AbsolutePath: fileName,
		FileName:     fileName,
		Sha256Hash:   sha256Hex(content),
		Size:         int64(len(content)),
	}
	asset, err := m.app.WriteQueries.InsertNoteAsset(ctx, assetParams)
	require.NoError(t, err)

	err = m.app.Storage.PutAssetObject(ctx, strings.NewReader(content), asset)
	require.NoError(t, err)

	linkParams := db.UpsertNoteVersionAssetParams{AssetID: asset.ID, VersionID: versionID, Path: fileName}
	err = m.app.WriteQueries.UpsertNoteVersionAsset(ctx, linkParams)
	require.NoError(t, err)

	m.assets[fileName] = model.AssetRoutePrefix + asset.Sha256Hash + "/" + fileName
}

func (m *freeNoteMatrix) seedReaders(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	a := m.app

	for i := range m.readers {
		reader := &m.readers[i]
		if reader.role == "" {
			continue
		}

		userParams := db.InsertUserWithEmailParams{Email: reader.name + "@example.com", CreatedVia: "test"}
		user, err := a.WriteQueries.InsertUserWithEmail(ctx, userParams)
		require.NoError(t, err)
		reader.userID = user.ID

		if reader.role == "admin" {
			_, err = a.WriteQueries.InsertAdmin(ctx, db.InsertAdminParams{UserID: user.ID})
			require.NoError(t, err)
		}

		for _, grant := range reader.grants {
			subgraph, sgErr := a.Queries.SubgraphByName(ctx, grant)
			require.NoError(t, sgErr)

			accessParams := db.CreateUserSubgraphAccessParams{UserID: user.ID, SubgraphID: subgraph.ID}
			_, err = a.WriteQueries.CreateUserSubgraphAccess(ctx, accessParams)
			require.NoError(t, err)
		}

		fctx := &fasthttp.RequestCtx{}
		stored, err := a.tokenManager.Store(fctx, usertoken.Data{ID: int(user.ID), Role: reader.role})
		require.NoError(t, err)
		m.cookies[reader.name] = stored.JWT
	}
}

type matrixResponse struct {
	status int
	body   string
}

func (m *freeNoteMatrix) do(t *testing.T, reader, method, uri string, body []byte, headers map[string]string) matrixResponse {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(method)
	fctx.Request.SetRequestURI(uri)
	fctx.Request.Header.SetHost("example.com")
	fctx.Request.Header.Set("Accept-Encoding", "gzip")
	for key, value := range headers {
		fctx.Request.Header.Set(key, value)
	}
	if cookie := m.cookies[reader]; cookie != "" {
		fctx.Request.Header.SetCookie("trip2g_token", cookie)
	}
	if body != nil {
		fctx.Request.SetBody(body)
	}

	m.handler(fctx)

	raw := fctx.Response.Body()
	if bytes.Equal(fctx.Response.Header.ContentEncoding(), []byte("gzip")) {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		require.NoError(t, err)
		raw, err = io.ReadAll(zr)
		require.NoError(t, err)
	}

	return matrixResponse{status: fctx.Response.StatusCode(), body: string(raw)}
}

func (m *freeNoteMatrix) markers(text string) string {
	var found []string
	for _, note := range m.notes {
		if strings.Contains(text, note.marker()) {
			found = append(found, note.key)
		}
	}
	return "[" + strings.Join(found, ",") + "]"
}

func (m *freeNoteMatrix) graphql(t *testing.T, reader, query string, variables map[string]interface{}) string {
	t.Helper()

	payload, err := json.Marshal(map[string]interface{}{"query": query, "variables": variables})
	require.NoError(t, err)

	headers := map[string]string{"Content-Type": "application/json"}
	resp := m.do(t, reader, http.MethodPost, "/_system/graphql", payload, headers)
	require.Equal(t, http.StatusOK, resp.status, resp.body)
	return resp.body
}

func (m *freeNoteMatrix) pageCell(t *testing.T, reader string, note matrixNote) string {
	t.Helper()

	resp := m.do(t, reader, http.MethodGet, note.permalink, nil, nil)
	wall := ""
	switch {
	case strings.Contains(resp.body, `id="signinwall"`):
		wall = ":signin"
	case strings.Contains(resp.body, "window.__trip2g_paywall"):
		wall = ":paywall"
	}
	return fmt.Sprintf("%d%s%s", resp.status, wall, m.markers(resp.body))
}

func (m *freeNoteMatrix) graphqlNoteCell(t *testing.T, reader string, note matrixNote) string {
	t.Helper()

	query := `query($path: String!) { note(input: {path: $path, referer: ""}) { html } }`
	body := m.graphql(t, reader, query, map[string]interface{}{"path": note.permalink})

	var out struct {
		Data struct {
			Note *struct {
				HTML string `json:"html"`
			} `json:"note"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	err := json.Unmarshal([]byte(body), &out)
	require.NoError(t, err)

	switch {
	case len(out.Errors) > 0:
		return "error"
	case out.Data.Note == nil:
		return "null"
	default:
		return m.markers(out.Data.Note.HTML)
	}
}

type matrixSearchResult struct {
	URL                string   `json:"url"`
	HighlightedContent []string `json:"highlightedContent"`
	Document           *struct {
		HTML string `json:"html"`
	} `json:"document"`
}

func (m *freeNoteMatrix) search(t *testing.T, reader string) []matrixSearchResult {
	t.Helper()

	query := `{ search(input: {query: "matrixword"}) { nodes { url highlightedContent document { ... on PublicNote { html } } } } }`
	body := m.graphql(t, reader, query, nil)

	var out struct {
		Data struct {
			Search struct {
				Nodes []matrixSearchResult `json:"nodes"`
			} `json:"search"`
		} `json:"data"`
	}
	err := json.Unmarshal([]byte(body), &out)
	require.NoError(t, err)
	return out.Data.Search.Nodes
}

func (m *freeNoteMatrix) searchCell(results []matrixSearchResult, note matrixNote) string {
	for _, result := range results {
		if !strings.HasSuffix(result.URL, note.permalink) {
			continue
		}
		text := strings.Join(result.HighlightedContent, " ")
		if result.Document != nil {
			text += result.Document.HTML
		}
		return m.markers(text)
	}
	return "-"
}

func (m *freeNoteMatrix) mcpCell(t *testing.T, reader string, note matrixNote) string {
	t.Helper()

	payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"note_html","arguments":{"path":%q}}}`, note.path)
	headers := map[string]string{"Content-Type": "application/json", "Accept": "application/json, text/event-stream"}
	resp := m.do(t, reader, http.MethodPost, "/_system/mcp", []byte(payload), headers)
	require.Equal(t, http.StatusOK, resp.status, resp.body)
	return m.markers(resp.body)
}

func (m *freeNoteMatrix) assetCell(t *testing.T, reader string, note matrixNote) string {
	t.Helper()

	if note.noAsset {
		return "-"
	}
	resp := m.do(t, reader, http.MethodHead, m.assets[note.asset()], nil, nil)
	return strconv.Itoa(resp.status)
}

func (m *freeNoteMatrix) cacheCell(t *testing.T, reader string, note matrixNote) string {
	t.Helper()

	m.app.ClearPageCache()
	m.do(t, reader, http.MethodGet, note.permalink, nil, nil)
	if m.app.pageCache.Len() > 0 {
		return "+"
	}
	return "-"
}

type matrixRSSItem struct {
	Link    string `xml:"link"`
	Encoded string `xml:"encoded"`
}

func (m *freeNoteMatrix) rss(t *testing.T, reader string) []matrixRSSItem {
	t.Helper()

	resp := m.do(t, reader, http.MethodGet, "/feed", nil, nil)
	require.Equal(t, http.StatusOK, resp.status, resp.body)

	var feed struct {
		Channel struct {
			Items []matrixRSSItem `xml:"item"`
		} `xml:"channel"`
	}
	err := xml.Unmarshal([]byte(resp.body), &feed)
	require.NoError(t, err, resp.body)
	return feed.Channel.Items
}

func (m *freeNoteMatrix) rssCell(items []matrixRSSItem, note matrixNote) string {
	for _, item := range items {
		if strings.HasSuffix(item.Link, note.permalink) {
			return m.markers(item.Encoded)
		}
	}
	return "-"
}

func (m *freeNoteMatrix) similar(t *testing.T, reader string) map[string]string {
	t.Helper()

	query := `{ similarNotes(input: {path: "hub.md", limit: 20}) { note { path html } } }`
	body := m.graphql(t, reader, query, nil)

	var out struct {
		Data struct {
			SimilarNotes []struct {
				Note struct {
					Path string `json:"path"`
					HTML string `json:"html"`
				} `json:"note"`
			} `json:"similarNotes"`
		} `json:"data"`
	}
	err := json.Unmarshal([]byte(body), &out)
	require.NoError(t, err)

	found := map[string]string{}
	for _, similar := range out.Data.SimilarNotes {
		found[similar.Note.Path] = m.markers(similar.Note.HTML)
	}
	return found
}

func yesNo(v bool) string {
	if v {
		return "+"
	}
	return "-"
}

func (m *freeNoteMatrix) table(t *testing.T) string {
	t.Helper()

	sitemap := m.do(t, "guest", http.MethodGet, "/sitemap.xml", nil, nil).body

	var lines []string
	for _, reader := range m.readers {
		searchResults := m.search(t, reader.name)
		rssItems := m.rss(t, reader.name)
		similar := m.similar(t, reader.name)
		hub := m.do(t, reader.name, http.MethodGet, "/hub", nil, nil).body

		for _, note := range m.notes {
			similarCell, ok := similar[note.permalink]
			if !ok {
				similarCell = "-"
			}

			line := fmt.Sprintf(
				"%-4s %-8s page=%s gql=%s search=%s mcp=%s asset=%s cache=%s rss=%s sitemap=%s backlink=%s similar=%s",
				note.key,
				reader.name,
				m.pageCell(t, reader.name, note),
				m.graphqlNoteCell(t, reader.name, note),
				m.searchCell(searchResults, note),
				m.mcpCell(t, reader.name, note),
				m.assetCell(t, reader.name, note),
				m.cacheCell(t, reader.name, note),
				m.rssCell(rssItems, note),
				yesNo(strings.Contains(sitemap, "<loc>https://example.com"+note.permalink+"</loc>")),
				yesNo(strings.Contains(hub, note.title())),
				similarCell,
			)
			lines = append(lines, line)
		}
	}

	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestFreeNoteAccessMatrix(t *testing.T) {
	m := seedFreeNoteMatrix(t)

	t.Run("every reader and path", func(t *testing.T) {
		got := m.table(t)
		t.Log("\n" + got)

		want := strings.TrimSpace(freeNoteAccessMatrix)
		require.Equal(t, want, got, pretty.Diff(strings.Split(want, "\n"), strings.Split(got, "\n")))
	})

	t.Run("page cache never crosses readers", func(t *testing.T) {
		m.requirePageCacheKeepsReadersApart(t)
	})
}

func (m *freeNoteMatrix) requirePageCacheKeepsReadersApart(t *testing.T) {
	t.Helper()

	cold := map[string]string{}
	for _, note := range m.notes {
		for _, reader := range m.readers {
			m.app.ClearPageCache()
			cold[note.key+" "+reader.name] = m.pageCell(t, reader.name, note)
		}
	}

	reversed := make([]matrixReader, 0, len(m.readers))
	for i := len(m.readers) - 1; i >= 0; i-- {
		reversed = append(reversed, m.readers[i])
	}

	for _, order := range [][]matrixReader{m.readers, reversed} {
		for _, note := range m.notes {
			m.app.ClearPageCache()
			for range 2 {
				for _, reader := range order {
					got := m.pageCell(t, reader.name, note)
					require.Equal(t, cold[note.key+" "+reader.name], got, "%s read by %s after a warm cache", note.key, reader.name)
				}
			}
		}
	}
}

const freeNoteAccessMatrix = `
n1   admin    page=200[n1] gql=[n1] search=[n1] mcp=[n1] asset=200 cache=- rss=[n1] sitemap=+ backlink=+ similar=[n1]
n1   guest    page=200[n1] gql=[n1] search=[n1] mcp=[n1] asset=200 cache=+ rss=[n1] sitemap=+ backlink=+ similar=[n1]
n1   insider  page=200[n1] gql=[n1] search=[n1] mcp=[n1] asset=200 cache=- rss=[n1] sitemap=+ backlink=+ similar=[n1]
n1   member   page=200[n1] gql=[n1] search=[n1] mcp=[n1] asset=200 cache=- rss=[n1] sitemap=+ backlink=+ similar=[n1]
n1   outsider page=200[n1] gql=[n1] search=[n1] mcp=[n1] asset=200 cache=- rss=[n1] sitemap=+ backlink=+ similar=[n1]
n10  admin    page=200[n10] gql=[n10] search=[n10] mcp=[n10] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n10]
n10  guest    page=200[n10] gql=[n10] search=[n10] mcp=[n10] asset=200 cache=+ rss=- sitemap=- backlink=+ similar=[n10]
n10  insider  page=200[n10] gql=[n10] search=[n10] mcp=[n10] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n10]
n10  member   page=200[n10] gql=[n10] search=[n10] mcp=[n10] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n10]
n10  outsider page=200[n10] gql=[n10] search=[n10] mcp=[n10] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n10]
n2   admin    page=200[n2] gql=[n2] search=[n2] mcp=[n2] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n2]
n2   guest    page=401:signin[] gql=error search=[] mcp=[] asset=401 cache=- rss=- sitemap=- backlink=+ similar=-
n2   insider  page=200[n2] gql=[n2] search=[n2] mcp=[n2] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n2]
n2   member   page=200[n2] gql=[n2] search=[n2] mcp=[n2] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n2]
n2   outsider page=200[n2] gql=[n2] search=[n2] mcp=[n2] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n2]
n3   admin    page=200[n3] gql=[n3] search=[n3] mcp=[n3] asset=200 cache=- rss=[n3] sitemap=+ backlink=+ similar=[n3]
n3   guest    page=200[n3] gql=[n3] search=[n3] mcp=[n3] asset=200 cache=+ rss=[n3] sitemap=+ backlink=+ similar=[n3]
n3   insider  page=200[n3] gql=[n3] search=[n3] mcp=[n3] asset=200 cache=- rss=[n3] sitemap=+ backlink=+ similar=[n3]
n3   member   page=200[n3] gql=[n3] search=[n3] mcp=[n3] asset=200 cache=- rss=[n3] sitemap=+ backlink=+ similar=[n3]
n3   outsider page=200[n3] gql=[n3] search=[n3] mcp=[n3] asset=200 cache=- rss=[n3] sitemap=+ backlink=+ similar=[n3]
n4   admin    page=200[n4] gql=[n4] search=[n4] mcp=[n4] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n4]
n4   guest    page=401:paywall[] gql=error search=[] mcp=[] asset=401 cache=- rss=- sitemap=- backlink=+ similar=-
n4   insider  page=200[n4] gql=[n4] search=[n4] mcp=[n4] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n4]
n4   member   page=403:paywall[] gql=error search=[] mcp=[] asset=403 cache=- rss=- sitemap=- backlink=+ similar=-
n4   outsider page=403:paywall[] gql=error search=[] mcp=[] asset=403 cache=- rss=- sitemap=- backlink=+ similar=-
n5   admin    page=200[n5] gql=[n5] search=[n5] mcp=[n5] asset=200 cache=- rss=[n5] sitemap=+ backlink=+ similar=[n5]
n5   guest    page=200[n5] gql=[n5] search=[n5] mcp=[n5] asset=200 cache=+ rss=[n5] sitemap=+ backlink=+ similar=[n5]
n5   insider  page=200[n5] gql=[n5] search=[n5] mcp=[n5] asset=200 cache=- rss=[n5] sitemap=+ backlink=+ similar=[n5]
n5   member   page=200[n5] gql=[n5] search=[n5] mcp=[n5] asset=200 cache=- rss=[n5] sitemap=+ backlink=+ similar=[n5]
n5   outsider page=200[n5] gql=[n5] search=[n5] mcp=[n5] asset=200 cache=- rss=[n5] sitemap=+ backlink=+ similar=[n5]
n6   admin    page=200[n6] gql=[n6] search=[n6] mcp=[n6] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n6]
n6   guest    page=401:paywall[] gql=error search=[] mcp=[] asset=401 cache=- rss=- sitemap=- backlink=+ similar=-
n6   insider  page=200[n6] gql=[n6] search=[n6] mcp=[n6] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n6]
n6   member   page=403:paywall[] gql=error search=[] mcp=[] asset=403 cache=- rss=- sitemap=- backlink=+ similar=-
n6   outsider page=200[n6] gql=[n6] search=[n6] mcp=[n6] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n6]
n7   admin    page=404[] gql=error search=- mcp=[] asset=200 cache=- rss=- sitemap=- backlink=- similar=-
n7   guest    page=404[] gql=error search=- mcp=[] asset=401 cache=- rss=- sitemap=- backlink=- similar=-
n7   insider  page=404[] gql=error search=- mcp=[] asset=403 cache=- rss=- sitemap=- backlink=- similar=-
n7   member   page=404[] gql=error search=- mcp=[] asset=403 cache=- rss=- sitemap=- backlink=- similar=-
n7   outsider page=404[] gql=error search=- mcp=[] asset=403 cache=- rss=- sitemap=- backlink=- similar=-
n8   admin    page=404[] gql=error search=- mcp=[] asset=- cache=- rss=- sitemap=- backlink=- similar=-
n8   guest    page=404[] gql=error search=- mcp=[] asset=- cache=- rss=- sitemap=- backlink=- similar=-
n8   insider  page=404[] gql=error search=- mcp=[] asset=- cache=- rss=- sitemap=- backlink=- similar=-
n8   member   page=404[] gql=error search=- mcp=[] asset=- cache=- rss=- sitemap=- backlink=- similar=-
n8   outsider page=404[] gql=error search=- mcp=[] asset=- cache=- rss=- sitemap=- backlink=- similar=-
n9a  admin    page=200[n3,n9a] gql=[n3,n9a] search=[n3,n9a] mcp=[n3,n9a] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n3,n9a]
n9a  guest    page=401:paywall[] gql=error search=[] mcp=[] asset=401 cache=- rss=- sitemap=- backlink=+ similar=-
n9a  insider  page=200[n3,n9a] gql=[n3,n9a] search=[n3,n9a] mcp=[n3,n9a] asset=200 cache=- rss=- sitemap=- backlink=+ similar=[n3,n9a]
n9a  member   page=403:paywall[] gql=error search=[] mcp=[] asset=403 cache=- rss=- sitemap=- backlink=+ similar=-
n9a  outsider page=403:paywall[] gql=error search=[] mcp=[] asset=403 cache=- rss=- sitemap=- backlink=+ similar=-
n9b  admin    page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] mcp=[n4,n9b] asset=200 cache=- rss=[n4,n9b] sitemap=+ backlink=+ similar=[n4,n9b]
n9b  guest    page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] mcp=[n4,n9b] asset=200 cache=+ rss=[n4,n9b] sitemap=+ backlink=+ similar=[n4,n9b]
n9b  insider  page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] mcp=[n4,n9b] asset=200 cache=- rss=[n4,n9b] sitemap=+ backlink=+ similar=[n4,n9b]
n9b  member   page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] mcp=[n4,n9b] asset=200 cache=- rss=[n4,n9b] sitemap=+ backlink=+ similar=[n4,n9b]
n9b  outsider page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] mcp=[n4,n9b] asset=200 cache=- rss=[n4,n9b] sitemap=+ backlink=+ similar=[n4,n9b]
n9c  admin    page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] mcp=[n2,n9c] asset=200 cache=- rss=[n2,n9c] sitemap=+ backlink=+ similar=[n2,n9c]
n9c  guest    page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] mcp=[n2,n9c] asset=200 cache=+ rss=[n2,n9c] sitemap=+ backlink=+ similar=[n2,n9c]
n9c  insider  page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] mcp=[n2,n9c] asset=200 cache=- rss=[n2,n9c] sitemap=+ backlink=+ similar=[n2,n9c]
n9c  member   page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] mcp=[n2,n9c] asset=200 cache=- rss=[n2,n9c] sitemap=+ backlink=+ similar=[n2,n9c]
n9c  outsider page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] mcp=[n2,n9c] asset=200 cache=- rss=[n2,n9c] sitemap=+ backlink=+ similar=[n2,n9c]
`
