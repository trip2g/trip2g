package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/federationhandoff"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func readerHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// site is one instance as a browser sees it: a handler behind a host.
type site struct {
	*store
	url  string
	host string
}

func newSite(t *testing.T, name string) *site {
	t.Helper()
	publicURL := "http://" + name + ".test"
	s := newStore(t, publicURL)
	s.app.config.FederationAcceptGuests = true
	return &site{store: s, url: publicURL, host: name + ".test"}
}

// edgeSecret is the shared secret of the edge kid, derived so each edge differs.
func edgeSecret(kid string) []byte {
	return []byte(readerHash("secret of " + kid)[:32])
}

// pairEdge connects hub to peer under kid: the hub's outbound key for the
// peer's MCP URL, and the peer's inbound key granting subgraphs.
func pairEdge(t *testing.T, hub, peer *site, kid string, subgraphs ...string) int64 {
	t.Helper()
	ctx := context.Background()
	kbURL := peer.url + "/_system/mcp"

	hubCrypt, err := hub.app.EncryptData(edgeSecret(kid))
	require.NoError(t, err)
	_, err = hub.app.WriteQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid: kid, SecretCrypt: hubCrypt, KbUrl: &kbURL, CreatedBy: hub.adminID,
	})
	require.NoError(t, err)

	peerCrypt, err := peer.app.EncryptData(edgeSecret(kid))
	require.NoError(t, err)
	inbound, err := peer.app.WriteQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid: kid, SecretCrypt: peerCrypt, CreatedBy: peer.adminID,
	})
	require.NoError(t, err)

	for _, name := range subgraphs {
		sg, sgErr := peer.app.Queries.SubgraphByName(ctx, name)
		require.NoError(t, sgErr)
		grant := db.InsertFederationSecretSubgraphParams{Kid: kid, SubgraphID: sg.ID, CreatedBy: peer.adminID}
		require.NoError(t, peer.app.WriteQueries.InsertFederationSecretSubgraph(ctx, grant))
	}
	return inbound.ID
}

func (s *site) signIn(t *testing.T, email string) string {
	t.Helper()
	user, err := s.app.WriteQueries.InsertUserWithEmail(context.Background(), db.InsertUserWithEmailParams{Email: email, CreatedVia: "test"})
	require.NoError(t, err)
	return s.sessionFor(t, user.ID, "user")
}

func (s *site) sessionFor(t *testing.T, userID int64, role string) string {
	t.Helper()
	stored, err := s.app.tokenManager.Store(&fasthttp.RequestCtx{}, usertoken.Data{ID: int(userID), Role: role})
	require.NoError(t, err)
	return stored.JWT
}

// browser keeps one session cookie per host and sends it the way SameSite=Lax
// does: on every same-site request and on a cross-site top-level GET, never on
// a cross-site POST.
type browser struct {
	cookies map[string]string
}

func newBrowser() *browser {
	return &browser{cookies: map[string]string{}}
}

type reply struct {
	status   int
	body     string
	location string
	cookie   string
	referrer string
}

// request is one request the browser makes. from is the origin of the page
// that made it: the browser sends it as Origin on a POST, and a POST from
// another site goes without the cookie. Empty from is a typed URL or a link
// followed on the same site; "null" is an opaque origin.
type request struct {
	method string
	uri    string
	form   url.Values
	from   string
}

func (b *browser) get(t *testing.T, s *site, uri string) reply {
	t.Helper()
	return b.send(t, s, request{method: http.MethodGet, uri: uri})
}

func (b *browser) post(t *testing.T, s *site, uri string, form url.Values, from string) reply {
	t.Helper()
	return b.send(t, s, request{method: http.MethodPost, uri: uri, form: form, from: from})
}

func (b *browser) send(t *testing.T, s *site, req request) reply {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(req.method)
	fctx.Request.SetRequestURI(req.uri)
	fctx.Request.Header.SetHost(s.host)
	if req.form != nil {
		fctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
		fctx.Request.SetBodyString(req.form.Encode())
	}
	if req.method == http.MethodPost && req.from != "" {
		fctx.Request.Header.Set("Origin", req.from)
	}
	crossSite := req.from != "" && req.from != s.url
	if cookie := b.cookies[s.host]; cookie != "" && (!crossSite || req.method == http.MethodGet) {
		fctx.Request.Header.SetCookie("trip2g_token", cookie)
	}

	s.handler(fctx)

	r := reply{
		status:   fctx.Response.StatusCode(),
		body:     string(fctx.Response.Body()),
		location: string(fctx.Response.Header.Peek("Location")),
		referrer: string(fctx.Response.Header.Peek("Referrer-Policy")),
	}
	if raw := fctx.Response.Header.PeekCookie("trip2g_token"); raw != nil {
		c := fasthttp.AcquireCookie()
		require.NoError(t, c.ParseBytes(raw))
		r.cookie = string(c.Value())
		if r.cookie == "" {
			delete(b.cookies, s.host)
		} else {
			b.cookies[s.host] = r.cookie
		}
		fasthttp.ReleaseCookie(c)
	}
	return r
}

var formField = regexp.MustCompile(`name="(token|path)" value="([^"]*)"`)
var formAction = regexp.MustCompile(`<form id="handoff" method="post" action="([^"]*)"`)

// pressButton is the KB-note button: a same-site POST to the hub, whose page
// posts the handoff to the peer.
func (b *browser) pressButton(t *testing.T, hub *site, notePath string) (string, url.Values) {
	t.Helper()

	r := b.post(t, hub, federationhandoff.Path, url.Values{"note": {notePath}}, hub.url)
	require.Equal(t, http.StatusOK, r.status, r.body)
	// Under no-referrer a browser posts the form with Origin: null.
	require.Equal(t, "strict-origin", r.referrer)
	require.NotContains(t, r.body, "<meta name=\"referrer\"", "a referrer meta could make the form's Origin null")
	require.NotContains(t, r.body, "sandbox", "a sandboxed frame posts with Origin: null")

	action := formAction.FindStringSubmatch(r.body)
	require.NotNil(t, action, r.body)
	form := url.Values{}
	for _, m := range formField.FindAllStringSubmatch(r.body, -1) {
		form.Set(m[1], m[2])
	}
	return action[1], form
}

// arrive posts a handoff to the peer from the page at origin from and follows
// the 303 with a top-level GET, which is the request that may set the cookie.
func (b *browser) arrive(t *testing.T, peer *site, form url.Values, from string) reply {
	t.Helper()

	posted := b.post(t, peer, federationguest.Path, form, from)
	if posted.status != http.StatusSeeOther {
		return posted
	}
	require.True(t, strings.HasPrefix(posted.location, federationguest.Path+"?"), posted.location)
	return b.send(t, peer, request{method: http.MethodGet, uri: posted.location, from: from})
}

func (b *browser) openKB(t *testing.T, hub, peer *site, notePath string) reply {
	t.Helper()

	action, form := b.pressButton(t, hub, notePath)
	require.Equal(t, peer.url+federationguest.Path, action, "the handoff goes to the peer the KB-note names")
	return b.arrive(t, peer, form, hub.url)
}

// hubOrigin is the origin handoffToken signs by default.
const hubOrigin = "http://hub.test"

// handoffToken signs a guest handoff by hand, so a test can say what a hub
// holding the edge secret could send, including what no hub of ours would.
func handoffToken(t *testing.T, kid string, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": kid})
	require.NoError(t, err)
	body := map[string]any{
		"iss": "http://hub.test", "iat": time.Now().Unix(), "exp": time.Now().Add(30 * time.Second).Unix(),
		"rid": readerHash(strconv.FormatInt(time.Now().UnixNano(), 10)), "use": "guest", "path": "/", "hop": 1,
		"sub": readerHash("someone@example.com"), "org": hubOrigin,
	}
	for k, v := range claims {
		body[k] = v
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, edgeSecret(kid))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func handoffForm(token string) url.Values {
	return url.Values{"token": {token}, "path": {"/"}}
}

// twoSites is a hub whose KB-note points at a peer holding notes in three
// subgraphs, paired so the edge grants only "shared".
type twoSites struct {
	hub, peer *site
	edgeID    int64
	reader    string
}

const hubKBNote = "peer.md"

func seedTwoSites(t *testing.T) *twoSites {
	t.Helper()
	ctx := context.Background()

	peer := newSite(t, "peer")
	for _, name := range []string{"shared", "other"} {
		require.NoError(t, peer.app.WriteQueries.InsertSubgraph(ctx, name))
	}
	insertLatestNote(t, peer.app, "index.md", "---\nfree: true\n---\nzzfrontzz\n")
	insertLatestNote(t, peer.app, "shared.md", "---\nsubgraph: shared\n---\nzzsharedzz\n")
	insertLatestNote(t, peer.app, "other.md", "---\nsubgraph: other\n---\nzzotherzz\n")
	peer.load(t)

	hub := newSite(t, "hub")
	kbFrontmatter := "mcp_federation_kb_url: " + peer.url + "/_system/mcp\n"
	insertLatestNote(t, hub.app, hubKBNote, "---\nfree: true\n"+kbFrontmatter+"mcp_federation_kb_id: peer\n---\nThe peer base.\n")
	insertLatestNote(t, hub.app, "closed-kb.md", "---\nmcp_federation_kb_url: "+peer.url+"/_system/mcp\n---\nA closed base.\n")
	hub.load(t)

	edgeID := pairEdge(t, hub, peer, "hub-to-peer", "shared")
	hub.signIn(t, "Reader@Example.com")
	return &twoSites{hub: hub, peer: peer, edgeID: edgeID, reader: readerHash("reader@example.com")}
}

func (s *twoSites) guest(t *testing.T) *browser {
	t.Helper()
	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	r := b.openKB(t, s.hub, s.peer, hubKBNote)
	require.Equal(t, http.StatusFound, r.status, r.body)
	require.Equal(t, "/", r.location)
	require.NotEmpty(t, b.cookies[s.peer.host])
	return b
}

func (s *twoSites) readerID(t *testing.T) int64 {
	t.Helper()
	user, err := s.hub.app.Queries.UserByEmail(context.Background(), "reader@example.com")
	require.NoError(t, err)
	return user.ID
}

func (b *browser) page(t *testing.T, s *site, path string) string {
	t.Helper()
	r := b.get(t, s, path)
	var marks []string
	for _, m := range []string{"zzfrontzz", "zzsharedzz", "zzotherzz"} {
		if strings.Contains(r.body, m) {
			marks = append(marks, strings.Trim(m, "z"))
		}
	}
	return fmt.Sprintf("%d%v", r.status, marks)
}

func TestTheKBButtonIsALinkForAnAnonymousReaderAndAHandoffForASignedInOne(t *testing.T) {
	s := seedTwoSites(t)

	anonymous := newBrowser().get(t, s.hub, "/peer")
	require.Equal(t, http.StatusOK, anonymous.status)
	require.Contains(t, anonymous.body, `<a href="http://peer.test/" class="tg-button" rel="noopener">Open knowledge base</a>`)
	require.NotContains(t, anonymous.body, federationhandoff.Path)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	signedIn := b.get(t, s.hub, "/peer")
	require.Equal(t, http.StatusOK, signedIn.status)
	require.Contains(t, signedIn.body, `<form class="kb-open" method="post" action="/_system/federation/handoff">`)
	require.Contains(t, signedIn.body, `<input type="hidden" name="note" value="peer.md">`)

	closed := newBrowser().get(t, s.hub, "/closed-kb")
	require.NotEqual(t, http.StatusOK, closed.status, "an unreadable KB-note is not shown")
	require.NotContains(t, closed.body, "Open knowledge base")

	_, form := b.pressButton(t, s.hub, hubKBNote)
	require.NotEmpty(t, form.Get("token"))
	refused := newBrowser().post(t, s.hub, federationhandoff.Path, url.Values{"note": {hubKBNote}}, s.hub.url)
	require.Equal(t, http.StatusForbidden, refused.status, "an anonymous reader gets no handoff")
	refused = b.post(t, s.hub, federationhandoff.Path, url.Values{"note": {"closed-kb.md"}}, s.hub.url)
	require.Equal(t, http.StatusForbidden, refused.status, "a reader who cannot read the KB-note gets no handoff")
}
