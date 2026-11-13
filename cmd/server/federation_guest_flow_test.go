package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/federationhandoff"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/usertoken"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestAGuestReadsTheEdgeSubgraphsAndNothingElse(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	got := []string{
		"/ " + b.page(t, s.peer, "/"),
		"/shared " + b.page(t, s.peer, "/shared"),
		"/other " + b.page(t, s.peer, "/other"),
	}
	want := []string{"/ 200[front]", "/shared 200[shared]", "/other 403[]"}
	require.Equal(t, want, got)
}

func TestAGuestPageViewIsRecordedUnderTheSyntheticUserOfTheEdge(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	createdVia := "federation:hub-to-peer:" + s.reader
	user, err := s.peer.app.Queries.FederationGuestUser(context.Background(), createdVia)
	require.NoError(t, err)
	require.Nil(t, user.Email)
	require.Nil(t, user.TgUserID)

	require.Eventually(t, func() bool {
		var views int
		row := s.peer.app.conn.QueryRow(`select count(*) from user_note_views v
			join users u on u.id = v.user_id
			join note_versions nv on nv.id = v.version_id
			join note_paths np on np.id = nv.path_id
			where u.created_via = ? and np.value = 'shared.md'`, createdVia)
		return row.Scan(&views) == nil && views == 1
	}, 5*time.Second, 20*time.Millisecond, "the view lands under the guest of the edge")

	again := newBrowser()
	again.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	require.Equal(t, http.StatusFound, again.openKB(t, s.hub, s.peer, hubKBNote).status)
	var guests int
	row := s.peer.app.conn.QueryRow(`select count(*) from users where created_via = ?`, createdVia)
	require.NoError(t, row.Scan(&guests))
	require.Equal(t, 1, guests, "the same reader over the same edge is one guest")
}

func TestAGuestCannotWrite(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	mutations := []string{
		`requestEmailSignInCode(input: {email: "x@example.com"}) { __typename }`,
		`signInByEmail(input: {email: "admin@example.com", code: "111111"}) { __typename }`,
		`createPaymentLink(input: {paymentType: CRYPTO, offerId: "1", returnPath: "/"}) { __typename }`,
		`createEmailWaitListRequest(input: {email: "x@example.com", pathId: 1}) { __typename }`,
		`submitForm(input: {noteVersionId: 1, fields: []}) { __typename }`,
		`toggleFavoriteNote(input: {pathId: 1, value: true}) { __typename }`,
		`hideNotes(input: {paths: ["shared.md"]}) { __typename }`,
		`commitNotes { __typename }`,
		`generateTgAttachCode(input: {botId: 1}) { __typename }`,
		`createUserToken(input: {name: "mine"}) { __typename }`,
		`revokeUserToken(input: {id: "1"}) { __typename }`,
		`admin { createUser(input: {email: "x@example.com"}) { __typename } }`,
		`signOut { __typename } toggleFavoriteNote(input: {pathId: 1, value: true}) { __typename }`,
	}
	for _, mutation := range mutations {
		body, err := json.Marshal(map[string]string{"query": "mutation { " + mutation + " }"})
		require.NoError(t, err)
		r := b.sendJSON(t, s.peer, "/_system/graphql", body)
		require.Equal(t, http.StatusOK, r.status, r.body)
		require.Contains(t, r.body, "a federation guest can only read", mutation)
	}

	for _, path := range []string{"/_system/mcp", "/_system/hat", "/shared", "/_system/sync"} {
		r := b.sendJSON(t, s.peer, path, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		require.Equal(t, http.StatusForbidden, r.status, path)
	}

	var userTokens, favorites int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from user_tokens`).Scan(&userTokens))
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from user_favorite_notes`).Scan(&favorites))
	require.Zero(t, userTokens+favorites)

	body, err := json.Marshal(map[string]string{"query": "mutation { signOut { __typename } }"})
	require.NoError(t, err)
	r := b.sendJSON(t, s.peer, "/_system/graphql", body)
	require.JSONEq(t, `{"data":{"signOut":{"__typename":"SignOutPayload"}}}`, r.body, "a guest can still leave")
}

func (b *browser) sendJSON(t *testing.T, s *site, path string, body []byte) reply {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodPost)
	fctx.Request.SetRequestURI(path)
	fctx.Request.Header.SetHost(s.host)
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.Header.Set("Accept", "application/json")
	fctx.Request.SetBody(body)
	fctx.Request.Header.SetCookie("trip2g_token", b.cookies[s.host])

	s.handler(fctx)

	r := reply{status: fctx.Response.StatusCode(), body: string(fctx.Response.Body())}
	if raw := fctx.Response.Header.PeekCookie("trip2g_token"); raw != nil {
		c := fasthttp.AcquireCookie()
		require.NoError(t, c.ParseBytes(raw))
		if len(c.Value()) == 0 {
			delete(b.cookies, s.host)
		}
		fasthttp.ReleaseCookie(c)
	}
	return r
}

func TestRevokingTheEdgeEndsTheGuestSession(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	require.NoError(t, s.peer.app.WriteQueries.RevokeFederationSecret(context.Background(), s.edgeID))

	require.Equal(t, "401[]", b.page(t, s.peer, "/shared"), "the next request is anonymous")
	require.Empty(t, b.cookies[s.peer.host], "and the cookie is cleared")
}

func TestTurningGuestsOffRefusesHandoffsAndEndsSessions(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	s.peer.app.config.FederationAcceptGuests = false

	require.Equal(t, "401[]", b.page(t, s.peer, "/shared"), "an open session stops working")
	require.Empty(t, b.cookies[s.peer.host])

	fresh := newBrowser()
	fresh.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	r := fresh.openKB(t, s.hub, s.peer, hubKBNote)
	require.Equal(t, http.StatusForbidden, r.status, "a new handoff is refused")
	require.Empty(t, fresh.cookies[s.peer.host])
}

func TestAReaderIDCannotLandInALocalAccount(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()

	// The hub's admin has the peer admin's email, so the hash the hub sends is
	// the hash of an existing local admin's address.
	hubAdmin := newBrowser()
	hubAdmin.cookies[s.hub.host] = s.hub.sessionFor(t, s.hub.adminID, "admin")
	r := hubAdmin.openKB(t, s.hub, s.peer, hubKBNote)
	require.Equal(t, http.StatusFound, r.status, r.body)

	session, err := s.peer.app.tokenManager.ParseToken(ctx, hubAdmin.cookies[s.peer.host])
	require.NoError(t, err)
	require.Equal(t, usertoken.RoleFederationGuest, session.Role)
	require.NotEqual(t, int(s.peer.adminID), session.ID)
	require.Equal(t, readerHash("admin@example.com"), session.Reader)

	body, err := json.Marshal(map[string]string{"query": "{ admin { allUsers { nodes { id } } } }"})
	require.NoError(t, err)
	adminReply := hubAdmin.sendJSON(t, s.peer, "/_system/graphql", body)
	require.NotContains(t, adminReply.body, `"nodes"`, adminReply.body)

	for _, reader := range []string{"admin@example.com", "1", strconv.FormatInt(s.peer.adminID, 10)} {
		token := handoffToken(t, "hub-to-peer", map[string]any{"sub": reader})
		refused := newBrowser().arrive(t, s.peer, handoffForm(token), hubOrigin)
		require.Equal(t, http.StatusForbidden, refused.status, "reader %q is not a hash", reader)
	}

	var admins int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from admins`).Scan(&admins))
	require.Equal(t, 1, admins)
}

func TestAnOwnSessionOnThePeerIsNotOverwritten(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	own := s.peer.signIn(t, "reader@example.com")
	b.cookies[s.peer.host] = own

	r := b.openKB(t, s.hub, s.peer, hubKBNote)
	require.Equal(t, http.StatusFound, r.status, r.body)
	require.Empty(t, r.cookie, "the GET leg sets no cookie")
	require.Equal(t, own, b.cookies[s.peer.host])

	session, err := s.peer.app.tokenManager.ParseToken(ctx, b.cookies[s.peer.host])
	require.NoError(t, err)
	require.Equal(t, "user", session.Role)
}

func TestTheHandoffGoesOneHopFurtherAndStopsAtTheLimit(t *testing.T) {
	ctx := context.Background()

	c := newSite(t, "c")
	require.NoError(t, c.app.WriteQueries.InsertSubgraph(ctx, "deep"))
	insertLatestNote(t, c.app, "index.md", "---\nfree: true\n---\nzzfrontzz\n")
	insertLatestNote(t, c.app, "deep.md", "---\nsubgraph: deep\n---\nzzsharedzz\n")
	c.load(t)

	b := newSite(t, "b")
	require.NoError(t, b.app.WriteQueries.InsertSubgraph(ctx, "shared"))
	insertLatestNote(t, b.app, "c.md", "---\nsubgraph: shared\nmcp_federation_kb_url: "+c.url+"/_system/mcp\n---\nThe c base.\n")
	b.load(t)

	a := newSite(t, "a")
	insertLatestNote(t, a.app, "b.md", "---\nfree: true\nmcp_federation_kb_url: "+b.url+"/_system/mcp\n---\nThe b base.\n")
	a.load(t)

	pairEdge(t, a, b, "a-to-b", "shared")
	pairEdge(t, b, c, "b-to-c", "deep")

	reader := newBrowser()
	reader.cookies[a.host] = a.signIn(t, "Walker@Example.com")
	require.Equal(t, http.StatusFound, reader.openKB(t, a, b, "b.md").status)

	page := reader.get(t, b, "/c")
	require.Equal(t, http.StatusOK, page.status)
	require.Contains(t, page.body, `action="/_system/federation/handoff"`, "the guest on b sees b's KB-note button")

	r := reader.openKB(t, b, c, "c.md")
	require.Equal(t, http.StatusFound, r.status, r.body)
	require.Equal(t, "200[shared]", reader.page(t, c, "/deep"))

	session, err := c.app.tokenManager.ParseToken(ctx, reader.cookies[c.host])
	require.NoError(t, err)
	require.Equal(t, 2, session.Hop)
	require.Equal(t, "b-to-c", session.Kid)
	require.Equal(t, readerHash("walker@example.com"), session.Reader, "b forwards the reader it received")

	_, form := reader.pressButton(t, b, "c.md")
	signedOrigin := verifiedOrigin(t, c, form.Get("token"))
	require.Equal(t, b.url, signedOrigin, "the second hop is signed with b's origin")
	fromA := newBrowser().arrive(t, c, form, a.url)
	require.Equal(t, http.StatusForbidden, fromA.status, "c checks the post against b, not the first hub")
	_, form = reader.pressButton(t, b, "c.md")

	c.app.config.MCPFederationMaxDepth = 1
	refused := newBrowser().arrive(t, c, form, b.url)
	require.Equal(t, http.StatusForbidden, refused.status, "c refuses a hop over its limit")

	b.app.config.MCPFederationMaxDepth = 1
	refusedAtB := reader.post(t, b, federationhandoff.Path, url.Values{"note": {"c.md"}}, b.url)
	require.Equal(t, http.StatusForbidden, refusedAtB.status, "b does not sign a hop over its own limit")
}

func verifiedOrigin(t *testing.T, peer *site, token string) string {
	t.Helper()
	claims, err := mcp.VerifyGuestHandoff(context.Background(), peer.app, token)
	require.NoError(t, err)
	return claims.Origin
}

func TestAHandoffPostedFromAnotherSiteIsRefused(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()

	reader := newBrowser()
	reader.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := reader.pressButton(t, s.hub, hubKBNote)
	require.Equal(t, s.hub.url, verifiedOrigin(t, s.peer, form.Get("token")))

	// A hub reader copies their own form onto another page; a victim's browser
	// posts it from there, or with no or an opaque origin.
	for _, from := range []string{"http://attacker.test", "", "null", "http://hub.test.attacker.test", "https://hub.test"} {
		victim := newBrowser()
		r := victim.arrive(t, s.peer, form, from)
		require.Equal(t, http.StatusForbidden, r.status, "posted from %q", from)
		require.Empty(t, r.location)
		require.Empty(t, r.cookie)
		require.Empty(t, victim.cookies)
	}

	_, err := s.peer.app.Queries.FederationGuestUser(ctx, "federation:hub-to-peer:"+s.reader)
	require.True(t, db.IsNoFound(err), "no guest user was looked up into existence: %v", err)

	forged := handoffForm(handoffToken(t, "hub-to-peer", map[string]any{"org": "http://attacker.test"}))
	forged.Set("token", forged.Get("token")[:len(forged.Get("token"))-4]+"AAAA")
	r := newBrowser().arrive(t, s.peer, forged, "http://attacker.test")
	require.Equal(t, http.StatusForbidden, r.status, "the origin claim cannot be changed without the edge secret")

	r = newBrowser().arrive(t, s.peer, form, s.hub.url)
	require.Equal(t, http.StatusFound, r.status, "the refusals did not spend the handoff, and the hub's own page still works")
}

func TestATicketOpensOnlyInTheBrowserThatPostedTheHandoff(t *testing.T) {
	s := seedTwoSites(t)

	// The attacker is a hub reader: they post their own handoff from the hub's
	// real page and keep the ticket URL instead of following it.
	attacker := newBrowser()
	attacker.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := attacker.pressButton(t, s.hub, hubKBNote)
	posted := attacker.post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusSeeOther, posted.status, posted.body)

	attrs := strings.ToLower(posted.nonceCookie)
	for _, attr := range []string{"httponly", "path=/_system/federation/guest", "max-age=30", "samesite=lax"} {
		require.Contains(t, attrs, attr, posted.nonceCookie)
	}
	require.NotContains(t, attrs, "secure", "insecure test config, the same condition as the session cookie")
	require.NotContains(t, posted.location, attacker.nonces[s.peer.host])

	victim := newBrowser()
	r := victim.get(t, s.peer, posted.location)
	require.Equal(t, http.StatusForbidden, r.status, "a ticket opened without the nonce cookie")
	require.Empty(t, r.cookie)
	require.Empty(t, victim.cookies)

	victim.nonces[s.peer.host] = "not-the-nonce"
	r = victim.get(t, s.peer, posted.location)
	require.Equal(t, http.StatusForbidden, r.status, "a ticket opened with another nonce")
	require.Empty(t, r.cookie)
	require.Empty(t, victim.nonces, "a failed attempt clears the nonce cookie too")

	r = attacker.get(t, s.peer, posted.location)
	require.Equal(t, http.StatusFound, r.status, "failed attempts elsewhere did not burn the ticket")
	require.NotEmpty(t, attacker.cookies[s.peer.host])
	require.Empty(t, attacker.nonces)

	r = attacker.get(t, s.peer, posted.location)
	require.Equal(t, http.StatusForbidden, r.status, "and it opens once")
}

func TestTheGuestEndpointRefuses(t *testing.T) {
	s := seedTwoSites(t)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := b.pressButton(t, s.hub, hubKBNote)
	require.Equal(t, http.StatusFound, newBrowser().arrive(t, s.peer, form, s.hub.url).status)

	handoff := mcp.GuestHandoff{Path: "/", Hop: 1, Reader: s.reader, Origin: hubOrigin}
	signed, err := mcp.SignGuestHandoff(edgeSecret("hub-to-peer"), "hub-to-peer", "http://hub.test", handoff)
	require.NoError(t, err)

	tests := []struct {
		name string
		form url.Values
	}{
		{name: "replayed", form: form},
		{name: "expired", form: handoffForm(handoffToken(t, "hub-to-peer", map[string]any{
			"iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(-30 * time.Second).Unix(),
		}))},
		{name: "for another path", form: url.Values{"token": {signed}, "path": {"/shared"}}},
		{name: "absolute target", form: url.Values{
			"token": {handoffToken(t, "hub-to-peer", map[string]any{"path": "https://evil.test/"})}, "path": {"https://evil.test/"},
		}},
		{name: "off-site target", form: url.Values{
			"token": {handoffToken(t, "hub-to-peer", map[string]any{"path": "//evil.test/"})}, "path": {"//evil.test/"},
		}},
		{name: "an MCP bearer", form: handoffForm(handoffToken(t, "hub-to-peer", map[string]any{"use": nil}))},
		{name: "unknown edge", form: handoffForm(handoffToken(t, "nobody", nil))},
		{name: "hop over the limit", form: handoffForm(handoffToken(t, "hub-to-peer", map[string]any{"hop": 4}))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newBrowser().arrive(t, s.peer, tt.form, hubOrigin)
			require.Equal(t, http.StatusForbidden, r.status, r.body)
			require.Empty(t, r.location)
			require.Empty(t, r.cookie)
		})
	}

	forged := newBrowser().get(t, s.peer, federationguest.Path+"?ticket=forged")
	require.Equal(t, http.StatusForbidden, forged.status)
}
