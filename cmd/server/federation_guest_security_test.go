package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"trip2g/internal/case/federationguest"
	"trip2g/internal/case/federationhandoff"
	"trip2g/internal/case/federationleave"
	"trip2g/internal/case/mcp"
	"trip2g/internal/db"
	"trip2g/internal/graph/generated"
	"trip2g/internal/router"
	"trip2g/internal/usertoken"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/vektah/gqlparser/v2/ast"
)

const evilKBNote = "evil.md"

func addEvilKBNote(t *testing.T, s *twoSites) {
	t.Helper()
	insertLatestNote(t, s.hub.app, evilKBNote,
		"---\nfree: true\nmcp_federation_kb_url: http://evil.test/_system/mcp\nmcp_federation_kb_id: peer\n---\nA note any hub writer can push.\n")
	s.hub.load(t)
}

func TestSecurityAKBNoteWithAnotherAddressGetsNoToken(t *testing.T) {
	s := seedTwoSites(t)
	addEvilKBNote(t, s)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	// Two notes share the kb_id, so which one the hub resolves the key from is
	// a map order: try enough times for either order to show.
	for range 24 {
		r := b.post(t, s.hub, federationhandoff.Path, url.Values{"note": {evilKBNote}}, s.hub.url)
		require.Equal(t, http.StatusForbidden, r.status, "no token is signed for a note that names another address: %s", r.body)
		require.NotContains(t, r.body, `name="token"`)
		require.NotContains(t, r.body, "evil.test")
	}

	_, form := b.pressButton(t, s.hub, hubKBNote)
	require.NotEmpty(t, form.Get("token"), "the real KB-note still works")
}

func TestSecurityAGuestSessionOfARecreatedKeyIsAnonymous(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	require.NoError(t, s.peer.app.WriteQueries.RevokeFederationSecret(ctx, s.edgeID))

	crypt, err := s.peer.app.EncryptData(edgeSecret("hub-to-peer"))
	require.NoError(t, err)
	_, err = s.peer.app.WriteQueries.InsertFederationSecret(ctx, db.InsertFederationSecretParams{
		Kid: "hub-to-peer", SecretCrypt: crypt, CreatedBy: s.peer.adminID,
	})
	require.NoError(t, err)
	s.peer.acceptGuests(t, "hub-to-peer", true)

	require.Equal(t, "401[]", b.page(t, s.peer, "/shared"), "a new key under the old kid does not revive the session")
	require.Empty(t, b.cookies[s.peer.host], "and the cookie is cleared")
}

func TestSecurityAGuestSessionForAnEdgeThatDoesNotExistIsAnonymous(t *testing.T) {
	s := seedTwoSites(t)

	stored, err := s.peer.app.tokenManager.Store(&fasthttp.RequestCtx{}, usertoken.Data{
		ID: int(s.peer.adminID), Role: usertoken.RoleFederationGuest, Kid: "ghost", Hop: 1, Reader: s.reader,
	})
	require.NoError(t, err)

	b := newBrowser()
	b.cookies[s.peer.host] = stored.JWT
	require.Equal(t, "401[]", b.page(t, s.peer, "/shared"))
	require.Empty(t, b.cookies[s.peer.host], "the cookie is cleared")
}

func TestSecurityAGuestCannotGoFurther(t *testing.T) {
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

	refused := reader.post(t, b, federationhandoff.Path, url.Values{"note": {"c.md"}}, b.url)
	require.Equal(t, http.StatusForbidden, refused.status, "a guest's KB-note button is refused by the base they are a guest on")
	require.NotContains(t, refused.body, `name="token"`)

	for _, hop := range []int{2, 3} {
		token := handoffToken(t, "b-to-c", map[string]any{"hop": hop, "org": b.url})
		r := newBrowser().arrive(t, c, handoffForm(token), b.url)
		require.Equal(t, http.StatusForbidden, r.status, "a token at hop %d", hop)
		require.Empty(t, r.cookie)
	}
}

func TestSecurityTheNonceCookieIsHostOnly(t *testing.T) {
	s := seedTwoSites(t)
	s.peer.app.config.UserToken.Insecure = false

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := b.pressButton(t, s.hub, hubKBNote)
	posted := b.post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusSeeOther, posted.status, posted.body)

	cookie := posted.nonceCookie
	require.True(t, strings.HasPrefix(cookie, "__Host-"), "a Secure nonce cookie takes the __Host- prefix: %s", cookie)
	require.Contains(t, cookie, nonceCookieSuffix, "named for the guest endpoint, not a generic name")
	attrs := strings.ToLower(cookie)
	for _, attr := range []string{"secure", "httponly", "samesite=lax", "max-age=30"} {
		require.Contains(t, attrs, attr, cookie)
	}
	require.Regexp(t, `path=/(;|$)`, attrs, "a __Host- cookie is at the root")
	require.NotContains(t, attrs, "domain=", "and has no Domain")

	require.Contains(t, b.nonces, s.peer.host, "a browser accepts it")
	entered := b.send(t, s.peer, request{method: http.MethodGet, uri: posted.location, from: s.hub.url})
	require.Equal(t, http.StatusFound, entered.status, entered.body)
	require.Contains(t, entered.nonceCookie, "2009 23:00:00 GMT", "the ticket leg clears it")
}

func TestSecurityTheSessionCookieIsHttpOnlyLaxAndSecureOnHTTPS(t *testing.T) {
	s := seedTwoSites(t)
	s.peer.app.tokenManager.SetInsecure(false)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	action, form := b.pressButton(t, s.hub, hubKBNote)
	require.Equal(t, s.peer.url+federationguest.Path, action)
	posted := b.post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusSeeOther, posted.status, posted.body)

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodGet)
	fctx.Request.SetRequestURI(posted.location)
	fctx.Request.Header.SetHost(s.peer.host)
	nonce := b.nonces[s.peer.host]
	fctx.Request.Header.SetCookie(nonce.name, nonce.value)
	s.peer.handler(fctx)
	require.Equal(t, http.StatusFound, fctx.Response.StatusCode())

	session := strings.ToLower(string(fctx.Response.Header.PeekCookie("trip2g_token")))
	for _, attr := range []string{"httponly", "samesite=lax", "secure"} {
		require.Contains(t, session, attr, session)
	}
}

func TestSecurityTheTicketIsOpaque(t *testing.T) {
	s := seedTwoSites(t)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := b.pressButton(t, s.hub, hubKBNote)
	posted := b.post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusSeeOther, posted.status, posted.body)

	parsed, err := url.Parse(posted.location)
	require.NoError(t, err)
	ticket := parsed.Query().Get(federationguest.TicketParam)
	require.NotEmpty(t, ticket)

	require.NotContains(t, posted.location, s.reader, "the reader's hash does not travel through logs and history")
	require.NotContains(t, posted.location, "hub-to-peer", "nor does the edge")
	require.Less(t, strings.Count(ticket, "."), 2, "the ticket is not a JWT")
	for _, part := range strings.Split(ticket, ".") {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(part)
		if decodeErr == nil {
			require.False(t, json.Valid(decoded), "the ticket decodes to nothing readable")
		}
	}

	entered := b.send(t, s.peer, request{method: http.MethodGet, uri: posted.location, from: s.hub.url})
	require.Equal(t, http.StatusFound, entered.status, entered.body)
	again := b.send(t, s.peer, request{method: http.MethodGet, uri: posted.location, from: s.hub.url})
	require.Equal(t, http.StatusForbidden, again.status, "an opaque ticket still opens once")
}

func TestSecurityTheHandoffPageIsNotStoredAndKeepsTheOrigin(t *testing.T) {
	s := seedTwoSites(t)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodPost)
	fctx.Request.SetRequestURI(federationhandoff.Path)
	fctx.Request.Header.SetHost(s.hub.host)
	fctx.Request.Header.Set("Origin", s.hub.url)
	fctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	fctx.Request.SetBodyString(url.Values{"note": {hubKBNote}}.Encode())
	fctx.Request.Header.SetCookie("trip2g_token", b.cookies[s.hub.host])
	s.hub.handler(fctx)

	require.Equal(t, http.StatusOK, fctx.Response.StatusCode())
	require.Equal(t, "no-store", string(fctx.Response.Header.Peek("Cache-Control")))
	require.Equal(t, "strict-origin", string(fctx.Response.Header.Peek("Referrer-Policy")))
}

func TestSecurityANotePageRefusesToBeFramed(t *testing.T) {
	s := seedTwoSites(t)

	for _, host := range []*site{s.hub, s.peer} {
		r := newBrowser().get(t, host, "/")
		require.Equal(t, http.StatusOK, r.status)
	}

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodGet)
	fctx.Request.SetRequestURI("/peer")
	fctx.Request.Header.SetHost(s.hub.host)
	s.hub.handler(fctx)
	require.Equal(t, http.StatusOK, fctx.Response.StatusCode())
	require.Contains(t, string(fctx.Response.Header.Peek("Content-Security-Policy")), "frame-ancestors 'self'",
		"a sibling subdomain cannot frame the KB-note and take the click")
}

func tamperedToken(t *testing.T, token, field string, value any) string {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	claims[field] = value
	raw, err = json.Marshal(claims)
	require.NoError(t, err)
	parts[1] = base64.RawURLEncoding.EncodeToString(raw)
	return strings.Join(parts, ".")
}

func TestSecurityEverySignedFieldOfTheHandoffIsBoundToTheSignature(t *testing.T) {
	s := seedTwoSites(t)

	signed := handoffToken(t, "hub-to-peer", nil)
	fields := map[string]any{
		"path": "/shared",
		"org":  "http://attacker.test",
		"hop":  2,
		"sub":  readerHash("someone-else@example.com"),
		"use":  "",
		"rid":  "another-id",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	for field, value := range fields {
		forged := tamperedToken(t, signed, field, value)
		form := handoffForm(forged)
		if field == "path" {
			form.Set("path", "/shared")
		}
		r := newBrowser().arrive(t, s.peer, form, hubOrigin)
		require.Equal(t, http.StatusForbidden, r.status, "changing %q after signing", field)
		require.Empty(t, r.cookie)
	}

	r := newBrowser().arrive(t, s.peer, handoffForm(signed), hubOrigin)
	require.Equal(t, http.StatusFound, r.status, "the untouched token is good")
}

func TestSecurityAHandoffIsBoundToItsEdgeAndItsLifetime(t *testing.T) {
	s := seedTwoSites(t)
	pairEdge(t, s.hub, s.peer, "other-edge", "other")

	good := handoffToken(t, "hub-to-peer", nil)
	parts := strings.Split(good, ".")
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": "other-edge"})
	require.NoError(t, err)
	reheaded := base64.RawURLEncoding.EncodeToString(header) + "." + parts[1] + "." + parts[2]

	now := time.Now()
	tests := []struct {
		name  string
		token string
	}{
		{"signed by a secret this edge does not share", handoffToken(t, "hub-to-peer", map[string]any{"iss": "x"}) + "AAAA"},
		{"another edge's header on this edge's signature", reheaded},
		{"signed under an edge's secret but naming another", func() string {
			other := handoffToken(t, "other-edge", nil)
			p := strings.Split(other, ".")
			h, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT", "kid": "hub-to-peer"})
			return base64.RawURLEncoding.EncodeToString(h) + "." + p[1] + "." + p[2]
		}()},
		{"expired", handoffToken(t, "hub-to-peer", map[string]any{"iat": now.Add(-2 * time.Minute).Unix(), "exp": now.Add(-90 * time.Second).Unix()})},
		{"issued in the future", handoffToken(t, "hub-to-peer", map[string]any{
			"iat": now.Add(time.Hour).Unix(), "exp": now.Add(time.Hour + 30*time.Second).Unix(),
		})},
		{"valid for longer than a handoff may live", handoffToken(t, "hub-to-peer", map[string]any{"exp": now.Add(time.Hour).Unix()})},
		{"an unknown edge", handoffToken(t, "nobody", nil)},
		{"no signature", strings.Join(parts[:2], ".") + "."},
		{"an unsigned header", func() string {
			h, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT", "kid": "hub-to-peer"})
			return base64.RawURLEncoding.EncodeToString(h) + "." + parts[1] + "."
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newBrowser().arrive(t, s.peer, handoffForm(tt.token), hubOrigin)
			require.Equal(t, http.StatusForbidden, r.status, r.body)
			require.Empty(t, r.cookie)
		})
	}
}

func TestSecurityAHandoffOpensOnce(t *testing.T) {
	s := seedTwoSites(t)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	_, form := b.pressButton(t, s.hub, hubKBNote)

	first := newBrowser().post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusSeeOther, first.status)
	second := newBrowser().post(t, s.peer, federationguest.Path, form, s.hub.url)
	require.Equal(t, http.StatusForbidden, second.status, "the same form posted twice")
	require.Empty(t, second.location)
}

func TestSecurityATargetThatLeavesTheSiteIsRefused(t *testing.T) {
	s := seedTwoSites(t)

	for _, target := range []string{
		"//evil.example", "https://evil.example", `/\evil.example`, "/ok\nSet-Cookie: a=b", "/ok\r\nLocation: //evil.example",
		"shared", "/\t/evil.example", "javascript:alert(1)",
	} {
		token := handoffToken(t, "hub-to-peer", map[string]any{"path": target})
		form := url.Values{"token": {token}, "path": {target}}
		r := newBrowser().arrive(t, s.peer, form, hubOrigin)
		require.Equal(t, http.StatusForbidden, r.status, "target %q", target)
		require.Empty(t, r.location, "target %q", target)
		require.Empty(t, r.cookie)
	}
}

func TestSecurityAGuestIsAnonymousAfterItsKeyLosesASubgraph(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	shared, err := s.peer.app.Queries.SubgraphByName(context.Background(), "shared")
	require.NoError(t, err)
	err = s.peer.app.WriteQueries.DeleteFederationSecretSubgraph(context.Background(), db.DeleteFederationSecretSubgraphParams{
		Kid: "hub-to-peer", SubgraphID: shared.ID,
	})
	require.NoError(t, err)

	require.Equal(t, "403[]", b.page(t, s.peer, "/shared"), "the notes of the removed subgraph are closed on the next request")
	require.Equal(t, "200[front]", b.page(t, s.peer, "/"), "free notes stay")
}

func TestSecurityABannedGuestIsRefusedBeforeAnythingIsShown(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	session, err := s.peer.app.tokenManager.ParseToken(context.Background(), b.cookies[s.peer.host])
	require.NoError(t, err)
	_, err = s.peer.app.conn.Exec(`insert into user_bans (user_id, banned_by, reason) values (?, ?, 'test')`, session.ID, s.peer.adminID)
	require.NoError(t, err)
	require.NoError(t, s.peer.app.UserBans.ResetBanCache(context.Background()))

	r := b.get(t, s.peer, "/shared")
	require.NotEqual(t, http.StatusOK, r.status)
	require.NotContains(t, r.body, "zzsharedzz")
}

func TestSecurityAGuestUserHasNoWayToSignIn(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	session, err := s.peer.app.tokenManager.ParseToken(context.Background(), b.cookies[s.peer.host])
	require.NoError(t, err)
	user, err := s.peer.app.Queries.UserByID(context.Background(), int64(session.ID))
	require.NoError(t, err)
	require.Nil(t, user.Email, "no email to send a code to")
	require.Nil(t, user.TgUserID, "no Telegram id to sign in with")
}

type guestGraphQL struct {
	t      *testing.T
	b      *browser
	peer   *site
	schema *ast.Schema
}

func (g guestGraphQL) literal(typ *ast.Type, depth int) string {
	if typ.Elem != nil {
		return "[" + g.literal(typ.Elem, depth+1) + "]"
	}
	def := g.schema.Types[typ.NamedType]
	switch def.Kind { //nolint:exhaustive // only enums and inputs need a literal of their own
	case ast.Enum:
		return def.EnumValues[0].Name
	case ast.InputObject:
		var fields []string
		for _, field := range def.Fields {
			if field.Type.NonNull && field.DefaultValue == nil && depth < 6 {
				fields = append(fields, field.Name+": "+g.literal(field.Type, depth+1))
			}
		}
		return "{" + strings.Join(fields, ", ") + "}"
	}
	switch typ.NamedType {
	case "Int", "Int64", "Float", "ID":
		return "1"
	case "Boolean":
		return "true"
	case "Time", "DateTime":
		return `"2026-01-01T00:00:00Z"`
	default:
		return `"x"`
	}
}

func (g guestGraphQL) call(field *ast.FieldDefinition) string {
	var args []string
	for _, arg := range field.Arguments {
		if arg.Type.NonNull {
			args = append(args, arg.Name+": "+g.literal(arg.Type, 0))
		}
	}
	call := field.Name
	if len(args) > 0 {
		call += "(" + strings.Join(args, ", ") + ")"
	}
	switch g.schema.Types[field.Type.Name()].Kind { //nolint:exhaustive // scalars and enums take no selection
	case ast.Object, ast.Union, ast.Interface:
		call += " { __typename }"
	}
	return call
}

func (g guestGraphQL) send(query string) string {
	g.t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(g.t, err)
	r := g.b.sendJSON(g.t, g.peer, "/_system/graphql", body)
	require.Equal(g.t, http.StatusOK, r.status, r.body)
	return r.body
}

func newGuestGraphQL(t *testing.T) (guestGraphQL, *twoSites) {
	t.Helper()
	s := seedTwoSites(t)
	return guestGraphQL{
		t: t, b: s.guest(t), peer: s.peer, schema: generated.NewExecutableSchema(generated.Config{}).Schema(),
	}, s
}

func TestSecurityEveryMutationInTheSchemaIsRefusedToAGuest(t *testing.T) {
	g, _ := newGuestGraphQL(t)

	roots := map[string]*ast.Definition{"": g.schema.Mutation}
	for _, field := range g.schema.Mutation.Fields {
		if field.Name == "admin" {
			roots["admin"] = g.schema.Types[field.Type.Name()]
		}
	}

	checked := 0
	for prefix, root := range roots {
		for _, field := range root.Fields {
			if strings.HasPrefix(field.Name, "__") || (prefix == "" && (field.Name == "signOut" || field.Name == "admin")) {
				continue
			}
			call := g.call(field)
			if prefix != "" {
				call = prefix + " { " + call + " }"
			}
			body := g.send("mutation { " + call + " }")
			require.Contains(t, body, "a federation guest can only read", "%s.%s: %s", prefix, field.Name, body)
			checked++
		}
	}
	require.Greater(t, checked, 100, "the list comes from the schema, and the schema has that many mutations")

	adminCall := g.call(g.schema.Mutation.Fields.ForName("admin"))
	body := g.send("mutation { " + adminCall + " }")
	require.Contains(t, body, "a federation guest can only read", "the admin namespace itself")
}

func TestSecurityNoAdminQueryAnswersAGuest(t *testing.T) {
	g, s := newGuestGraphQL(t)

	adminField := g.schema.Query.Fields.ForName("admin")
	require.NotNil(t, adminField)
	adminType := g.schema.Types[adminField.Type.Name()]

	adminBrowser := newBrowser()
	adminBrowser.cookies[s.peer.host] = s.peer.sessionFor(t, s.peer.adminID, "admin")
	adminGraphQL := guestGraphQL{t: t, b: adminBrowser, peer: s.peer, schema: g.schema}

	answeredToAdmin, checkedAsAdmin := 0, 0
	for _, field := range adminType.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}
		query := "{ admin { " + g.call(field) + " } }"

		var out struct {
			Data   map[string]map[string]any `json:"data"`
			Errors []any                     `json:"errors"`
		}
		require.NoError(t, json.Unmarshal([]byte(g.send(query)), &out), field.Name)
		require.NotEmpty(t, out.Errors, "admin.%s answers a guest: %v", field.Name, out.Data)
		require.Nil(t, out.Data["admin"], "admin.%s", field.Name)

		if checkedAsAdmin >= 12 {
			continue
		}
		checkedAsAdmin++

		var asAdmin struct {
			Data map[string]map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(adminGraphQL.send(query)), &asAdmin), field.Name)
		if asAdmin.Data["admin"] != nil {
			answeredToAdmin++
		}
	}
	require.Greater(t, answeredToAdmin, 3, "the same queries do answer an admin, so the refusals above mean something")
}

func TestSecurityNoNonGetEndpointAnswersAGuest(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	allowed := map[string]bool{
		federationhandoff.Path: true,
		federationguest.Path:   true,
		federationleave.Path:   true,
		"/_system/graphql":     true,
	}

	checked := 0
	for _, endpoint := range router.Endpoints() {
		if endpoint.Method() == http.MethodGet || endpoint.Path() == "" || allowed[endpoint.Path()] {
			continue
		}
		r := b.sendJSON(t, s.peer, endpoint.Path(), []byte(`{}`))
		require.Equal(t, http.StatusForbidden, r.status, "%s %s: %s", endpoint.Method(), endpoint.Path(), r.body)
		checked++
	}
	require.Greater(t, checked, 5, "the list comes from the router")

	for _, path := range []string{"/graphql", "/_system/graphql"} {
		body, err := json.Marshal(map[string]string{"query": `mutation { createUserToken(input: {name: "x"}) { __typename } }`})
		require.NoError(t, err)
		r := b.sendJSON(t, s.peer, path, body)
		if path == "/_system/graphql" {
			require.Contains(t, r.body, "a federation guest can only read")
		}
	}
}

func TestSecurityAGuestMayNotMintAnythingThatSignsIn(t *testing.T) {
	g, s := newGuestGraphQL(t)

	mutations := []string{
		`createUserToken(input: {name: "mine"}) { __typename }`,
		`requestEmailSignInCode(input: {email: "admin@example.com"}) { __typename }`,
		`signInByEmail(input: {email: "admin@example.com", code: "123456"}) { __typename }`,
	}
	adminMutation := g.schema.Types[g.schema.Mutation.Fields.ForName("admin").Type.Name()]
	for _, field := range adminMutation.Fields {
		for _, word := range []string{"ApiKey", "Hat", "Token", "SignIn"} {
			if strings.Contains(field.Name, word) {
				mutations = append(mutations, "admin { "+g.call(field)+" }")
				break
			}
		}
	}
	require.Greater(t, len(mutations), 4, "the schema has admin mutations that mint keys, tokens or links")

	for _, mutation := range mutations {
		body := g.send("mutation { " + mutation + " }")
		require.Contains(t, body, "a federation guest can only read", mutation)
	}

	var tokens, keys int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from user_tokens`).Scan(&tokens))
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from api_keys`).Scan(&keys))
	require.Zero(t, tokens+keys)
}

func TestSecurityAGuestReadsNothingOfASubgraphThatWasNotGranted(t *testing.T) {
	const marker = "zzcloseme"
	s := seedTwoSites(t)
	ctx := context.Background()

	versionID := insertLatestNote(t, s.peer.app, "closed.md",
		"---\nsubgraph: other\ntitle: Closed title\nsecret_field: "+marker+"meta\n---\n"+marker+"body ![[closed.png]]\n")
	s.peer.attach(t, versionID, "closed.png", marker+"asset")
	insertLatestNote(t, s.peer.app, "opens.md", "---\nfree: true\n---\nlinks to [[closed]]\n")
	s.peer.load(t)
	require.NoError(t, s.peer.app.WriteQueries.InsertSubgraph(ctx, "unused"))

	b := s.guest(t)
	g := guestGraphQL{t: t, b: b, peer: s.peer}

	answers := map[string]string{}
	answers["page"] = b.get(t, s.peer, "/closed").body
	answers["page latest"] = b.get(t, s.peer, "/closed?version=latest").body
	answers["page raw"] = b.get(t, s.peer, "/closed.md").body
	answers["opens"] = b.get(t, s.peer, "/opens").body
	answers["sitemap"] = b.get(t, s.peer, "/sitemap.xml").body
	answers["asset"] = b.get(t, s.peer, "/closed.png").body

	for name, query := range map[string]string{
		"note":    `{ note(input: {path: "/closed", referer: ""}) { html title } }`,
		"similar": `{ similarNotes(input: {path: "/shared"}) { __typename } }`,
		"search":  `{ search(input: {query: "` + marker + `"}) { nodes { url highlightedContent } } }`,
		"notes":   `{ notePaths { value content } }`,
	} {
		answers[name] = g.send(query)
	}

	for name, body := range answers {
		require.NotContains(t, body, marker, "%s leaks the closed note", name)
	}
}

func TestSecurityKnownExposuresAndTheClosedEmbed(t *testing.T) {
	s := seedTwoSites(t)
	insertLatestNote(t, s.peer.app, "closed.md", "---\nsubgraph: other\ntitle: Closed title\n---\nzzembeddedzz\n")
	insertLatestNote(t, s.peer.app, "opens.md", "---\nfree: true\n---\nlinks to [[closed]]\n")
	insertLatestNote(t, s.peer.app, "embeds.md", "---\nfree: true\n---\nembeds ![[closed]]\n")
	s.peer.load(t)
	b := s.guest(t)

	closed := b.get(t, s.peer, "/closed")
	require.Equal(t, http.StatusForbidden, closed.status)
	require.Contains(t, closed.body, "Closed title", "a closed note's title is shown on its closed page")

	opens := b.get(t, s.peer, "/opens")
	require.Equal(t, http.StatusOK, opens.status)
	require.Contains(t, opens.body, "paywall-other", "subgraph names are in the markup of links to closed notes")

	for name, reader := range map[string]*browser{"a guest": b, "an anonymous reader": newBrowser()} {
		embeds := reader.get(t, s.peer, "/embeds")
		require.Equal(t, http.StatusOK, embeds.status)
		require.NotContains(t, embeds.body, "zzembeddedzz", "%s: a free note that embeds a closed one shows a link, not the text", name)
		require.Contains(t, embeds.body, `href="/closed"`, "%s: the closed embed stays a link", name)
	}
}

type edgePair struct {
	hub, peer *site
}

func seedEdgePair(t *testing.T) *edgePair {
	t.Helper()
	ctx := context.Background()

	peer := newSite(t, "peer")
	require.NoError(t, peer.app.WriteQueries.InsertSubgraph(ctx, "shared"))
	insertLatestNote(t, peer.app, "index.md", "---\nfree: true\n---\nzzfrontzz\n")
	insertLatestNote(t, peer.app, "shared.md", "---\nsubgraph: shared\n---\nzzsharedzz\n")
	peer.load(t)

	hub := newSite(t, "hub")
	for _, kid := range []string{"edge-a", "edge-b"} {
		frontmatter := "---\nfree: true\nmcp_federation_kb_url: " + peer.url + "/_system/mcp\nmcp_federation_kb_id: " + kid + "\n---\n"
		insertLatestNote(t, hub.app, kid+".md", frontmatter+"A base.\n")
	}
	hub.load(t)
	hub.signIn(t, "Reader@Example.com")

	pairEdge(t, hub, peer, "edge-a", "shared")
	pairEdge(t, hub, peer, "edge-b", "shared")
	return &edgePair{hub: hub, peer: peer}
}

func (p *edgePair) reader(t *testing.T) *browser {
	t.Helper()
	user, err := p.hub.app.Queries.UserByEmail(context.Background(), "reader@example.com")
	require.NoError(t, err)
	b := newBrowser()
	b.cookies[p.hub.host] = p.hub.sessionFor(t, user.ID, "user")
	return b
}

func TestSecurityAcceptingGuestsIsASettingOfTheEdge(t *testing.T) {
	p := seedEdgePair(t)
	p.peer.acceptGuests(t, "edge-b", false)

	refused := p.reader(t).openKB(t, p.hub, p.peer, "edge-b.md")
	require.Equal(t, http.StatusForbidden, refused.status, "an edge that does not accept people refuses a handoff")
	require.Empty(t, refused.cookie)

	guestA := p.reader(t)
	require.Equal(t, http.StatusFound, guestA.openKB(t, p.hub, p.peer, "edge-a.md").status, "the edge that accepts admits")
	require.Equal(t, "200[shared]", guestA.page(t, p.peer, "/shared"))

	p.peer.acceptGuests(t, "edge-b", true)
	guestB := p.reader(t)
	require.Equal(t, http.StatusFound, guestB.openKB(t, p.hub, p.peer, "edge-b.md").status)
	require.Equal(t, "200[shared]", guestB.page(t, p.peer, "/shared"))

	p.peer.acceptGuests(t, "edge-a", false)
	require.Equal(t, "401[]", guestA.page(t, p.peer, "/shared"), "switching it off ends that edge's sessions on the next request")
	require.Empty(t, guestA.cookies[p.peer.host])
	require.Equal(t, "200[shared]", guestB.page(t, p.peer, "/shared"), "and leaves the other edge's alone")

	again := p.reader(t).openKB(t, p.hub, p.peer, "edge-a.md")
	require.Equal(t, http.StatusForbidden, again.status)
}

func TestSecurityAnEdgeThatNeverSwitchedPeopleOnAdmitsNobody(t *testing.T) {
	s := seedTwoSites(t)
	s.peer.acceptGuests(t, "hub-to-peer", false)

	r := newBrowser()
	r.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	require.Equal(t, http.StatusForbidden, r.openKB(t, s.hub, s.peer, hubKBNote).status)

	var accepting int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from federation_secrets where kb_url is null and accept_guests`).Scan(&accepting))
	require.Zero(t, accepting)

	crypt, err := s.peer.app.EncryptData(edgeSecret("fresh"))
	require.NoError(t, err)
	_, err = s.peer.app.WriteQueries.InsertFederationSecret(context.Background(), db.InsertFederationSecretParams{
		Kid: "fresh", SecretCrypt: crypt, CreatedBy: s.peer.adminID,
	})
	require.NoError(t, err)
	require.NoError(t, s.peer.app.conn.QueryRow(`select accept_guests from federation_secrets where kid = 'fresh'`).Scan(&accepting))
	require.Zero(t, accepting, "a new key starts with people off")
}

func TestSecurityTwoFirstHandoffsOfOneReaderLeaveOneRow(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()

	const racers = 12
	ids := make(chan int64, racers)
	errs := make(chan error, racers)
	for range racers {
		go func() {
			id, err := s.peer.app.FederationGuestUser(ctx, "hub-to-peer", s.reader)
			ids <- id
			errs <- err
		}()
	}

	first := <-ids
	require.NoError(t, <-errs)
	for range racers - 1 {
		require.Equal(t, first, <-ids, "every racer gets the same user")
		require.NoError(t, <-errs)
	}

	var rows, users int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from federation_guests`).Scan(&rows))
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from users where email is null and tg_user_id is null`).Scan(&users))
	require.Equal(t, 1, rows)
	require.Equal(t, 1, users, "and no orphan user is left behind")
}

func TestSecurityAGuestIsNotInTheAdminUserList(t *testing.T) {
	g, s := newGuestGraphQL(t)

	session, err := s.peer.app.tokenManager.ParseToken(context.Background(), g.b.cookies[s.peer.host])
	require.NoError(t, err)

	admin := newBrowser()
	admin.cookies[s.peer.host] = s.peer.sessionFor(t, s.peer.adminID, "admin")
	adminGraphQL := guestGraphQL{t: t, b: admin, peer: s.peer, schema: g.schema}
	body := adminGraphQL.send(`{ admin { allUsers { nodes { id } } } }`)

	var out struct {
		Data struct {
			Admin struct {
				AllUsers struct {
					Nodes []struct {
						ID json.Number `json:"id"`
					} `json:"nodes"`
				} `json:"allUsers"`
			} `json:"admin"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &out), body)
	require.NotEmpty(t, out.Data.Admin.AllUsers.Nodes, body)
	for _, node := range out.Data.Admin.AllUsers.Nodes {
		require.NotEqual(t, strconv.Itoa(session.ID), node.ID.String(), "the guest is a user with a row in federation_guests, and the list leaves those out")
	}
}

func TestSecurityAGuestSessionLivesTwelveHours(t *testing.T) {
	s := seedTwoSites(t)
	secret := []byte(s.peer.app.config.UserToken.Secret)

	forged := func(role string, age time.Duration, guestKey int64) string {
		claims := jwt.MapClaims{
			"i": s.peer.adminID, "r": role,
			"iat": time.Now().Add(-age).Unix(),
			"exp": time.Now().Add(24 * time.Hour).Unix(),
		}
		if role == usertoken.RoleFederationGuest {
			claims["i"] = s.peer.adminID + 100
			claims["k"] = "hub-to-peer"
			claims["ki"] = guestKey
			claims["h"] = 1
			claims["rd"] = s.reader
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
		require.NoError(t, err)
		return signed
	}

	key, err := s.peer.app.Queries.FederationSecretByKID(context.Background(), "hub-to-peer")
	require.NoError(t, err)

	young := newBrowser()
	young.cookies[s.peer.host] = forged(usertoken.RoleFederationGuest, 12*time.Hour-time.Minute, key.ID)
	require.Equal(t, "200[shared]", young.page(t, s.peer, "/shared"), "a guest session inside its twelve hours")

	old := newBrowser()
	old.cookies[s.peer.host] = forged(usertoken.RoleFederationGuest, 12*time.Hour+time.Second, key.ID)
	require.Equal(t, "401[]", old.page(t, s.peer, "/shared"), "a guest session minted twelve hours and a second ago is anonymous")

	ordinary := newBrowser()
	ordinary.cookies[s.peer.host] = forged("admin", 12*time.Hour+time.Second, 0)
	require.Equal(t, "200[shared]", ordinary.page(t, s.peer, "/shared"), "an ordinary session of the same age is not")
}

const openPath = "/_system/federation/open"

func (s *twoSites) openLink(note, path string) string {
	return openPath + "?" + url.Values{"note": {note}, "path": {path}}.Encode()
}

var openFormField = regexp.MustCompile(`name="(note|path)" value="([^"]*)"`)

func (b *browser) pressOpenButton(t *testing.T, hub *site, page reply) (string, url.Values) {
	t.Helper()
	form := url.Values{}
	for _, m := range openFormField.FindAllStringSubmatch(page.body, -1) {
		form.Set(m[1], html.UnescapeString(m[2]))
	}
	require.NotEmpty(t, form.Get("note"), page.body)
	r := b.post(t, hub, federationhandoff.Path, form, hub.url)
	require.Equal(t, http.StatusOK, r.status, r.body)
	action := formAction.FindStringSubmatch(r.body)
	require.NotNil(t, action, r.body)
	handoff := url.Values{}
	for _, m := range formField.FindAllStringSubmatch(r.body, -1) {
		handoff.Set(m[1], m[2])
	}
	return action[1], handoff
}

func TestSecurityAKBNoteNamesTheLandingPage(t *testing.T) {
	s := seedTwoSites(t)
	insertLatestNote(t, s.hub.app, "landing.md",
		"---\nfree: true\nmcp_federation_kb_url: "+s.peer.url+"/_system/mcp\nmcp_federation_kb_path: /shared\n---\nA base with a landing page.\n")
	insertLatestNote(t, s.hub.app, "bad-landing.md",
		"---\nfree: true\nmcp_federation_kb_url: "+s.peer.url+"/_system/mcp\nmcp_federation_kb_path: //evil.test/\n---\nA base with a bad landing page.\n")
	s.hub.load(t)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	landed := b.openKB(t, s.hub, s.peer, "landing.md")
	require.Equal(t, http.StatusFound, landed.status, landed.body)
	require.Equal(t, "/shared", landed.location, "the guest lands on the page the KB-note names")

	front := newBrowser()
	front.cookies[s.hub.host] = b.cookies[s.hub.host]
	require.Equal(t, "/", front.openKB(t, s.hub, s.peer, hubKBNote).location, "without the key, on the front page")

	refused := b.post(t, s.hub, federationhandoff.Path, url.Values{"note": {"bad-landing.md"}}, s.hub.url)
	require.Equal(t, http.StatusForbidden, refused.status, "a landing page that leaves the site is refused before signing")
	require.NotContains(t, refused.body, `name="token"`)
}

func TestSecurityTheOpenLinkLandsOnThePageAfterTheButton(t *testing.T) {
	s := seedTwoSites(t)
	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	page := b.get(t, s.hub, s.openLink(hubKBNote, "/shared"))
	require.Equal(t, http.StatusOK, page.status, page.body)
	require.Contains(t, page.body, "peer.test", "the page names the neighbour")
	require.Contains(t, page.body, `action="`+federationhandoff.Path+`"`)
	require.Equal(t, "no-store", page.cacheControl)
	require.Empty(t, b.cookies[s.peer.host], "the link alone opens nothing")

	action, form := b.pressOpenButton(t, s.hub, page)
	require.Equal(t, s.peer.url+federationguest.Path, action)
	landed := b.arrive(t, s.peer, form, s.hub.url)
	require.Equal(t, http.StatusFound, landed.status, landed.body)
	require.Equal(t, "/shared", landed.location)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))
}

func TestSecurityTheOpenLinkRefusesATargetThatLeavesTheSite(t *testing.T) {
	s := seedTwoSites(t)
	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	for _, target := range []string{"//evil.example", "https://evil.example/", "shared", `/\evil.example`, "/ok\nSet-Cookie: a=b", "/ok\r\nLocation: //evil.example"} {
		page := b.get(t, s.hub, s.openLink(hubKBNote, target))
		require.Equal(t, http.StatusForbidden, page.status, "open link path %q", target)
		require.NotContains(t, page.body, `name="token"`)

		posted := b.post(t, s.hub, federationhandoff.Path, url.Values{"note": {hubKBNote}, "path": {target}}, s.hub.url)
		require.Equal(t, http.StatusForbidden, posted.status, "handoff path %q is refused again before signing", target)
		require.NotContains(t, posted.body, `name="token"`)
	}
}

func TestSecurityTheOpenLinkAsksAnAnonymousReaderToSignInAndComesBack(t *testing.T) {
	s := seedTwoSites(t)

	link := s.openLink(hubKBNote, "/shared")
	anonymous := newBrowser().get(t, s.hub, link)
	require.Equal(t, http.StatusUnauthorized, anonymous.status, anonymous.body)
	require.Contains(t, anonymous.body, `id="signinwall"`, "the hub's own sign-in comes first")
	require.NotContains(t, anonymous.body, federationhandoff.Path)

	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")
	again := b.get(t, s.hub, link)
	require.Equal(t, http.StatusOK, again.status, "the same URL after signing in is the page with the button")
	require.Contains(t, again.body, `action="`+federationhandoff.Path+`"`)
}

func TestSecurityTheOpenLinkIsRefusedToAReaderWhoCannotReadTheKBNote(t *testing.T) {
	s := seedTwoSites(t)
	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	r := b.get(t, s.hub, s.openLink("closed-kb.md", "/"))
	require.Equal(t, http.StatusForbidden, r.status, r.body)
	require.NotContains(t, r.body, `name="note"`)

	r = b.get(t, s.hub, s.openLink("missing.md", "/"))
	require.Equal(t, http.StatusForbidden, r.status)
}

func TestSecurityFollowingTheOpenLinkFromAnotherSiteOpensNothing(t *testing.T) {
	s := seedTwoSites(t)
	b := newBrowser()
	b.cookies[s.hub.host] = s.hub.sessionFor(t, s.readerID(t), "user")

	r := b.send(t, s.hub, request{method: http.MethodGet, uri: s.openLink(hubKBNote, "/shared"), from: "http://attacker.test"})
	require.Equal(t, http.StatusOK, r.status)
	require.NotContains(t, r.body, `name="token"`, "the page waits for the button and signs nothing")
	require.NotContains(t, r.body, ".submit()", "and does not press it")

	var rows int
	require.NoError(t, s.peer.app.conn.QueryRow(`select count(*) from federation_guests`).Scan(&rows))
	require.Zero(t, rows)
	require.Empty(t, b.cookies[s.peer.host])
}

func TestSecurityAFederatedSearchResultCarriesAnOpenLinkThatWorks(t *testing.T) {
	ctx := context.Background()

	peerStore := newStore(t, "http://peer.invalid")
	require.NoError(t, peerStore.app.WriteQueries.InsertSubgraph(ctx, "shared"))
	insertLatestNote(t, peerStore.app, "index.md", "---\nfree: true\n---\nfront\n")
	insertLatestNote(t, peerStore.app, "skills/reel.md", "---\nsubgraph: shared\n---\nzzfindmezz in the reel.\n")
	peerStore.load(t)
	peerURL := serve(t, peerStore)
	peerStore.app.config.PublicURL = peerURL
	peer := &site{store: peerStore, url: peerURL, host: "peer.test"}

	hubStore := newStore(t, "http://hub.test")
	kbURL := peerURL + "/_system/mcp"
	insertLatestNote(t, hubStore.app, hubKBNote, "---\nfree: true\nmcp_federation_kb_url: "+kbURL+"\nmcp_federation_kb_id: peer\n---\nThe peer.\n")
	hubStore.load(t)
	hub := &site{store: hubStore, url: "http://hub.test", host: "hub.test"}
	edgeID := pairEdge(t, hub, peer, "hub-to-peer", "shared")
	require.NotZero(t, edgeID)
	hub.signIn(t, "Reader@Example.com")

	res := hubStore.tool(t, "federated_search", `{"kb_id":"peer","query":"zzfindmezz"}`)
	require.Nil(t, res.Error)
	result := res.Result.(mcp.CallToolResult)

	var payload mcp.SearchResultPayload
	require.NoError(t, json.Unmarshal(result.StructuredContent.(json.RawMessage), &payload))
	require.NotEmpty(t, payload.Results)
	link := payload.Results[0].OpenURL
	require.True(t, strings.HasPrefix(link, "http://hub.test"+openPath+"?"), link)
	require.Contains(t, result.Content[len(result.Content)-1].Text, link, "a text-only client sees it too")

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	require.Equal(t, hubKBNote, parsed.Query().Get("note"))
	require.Equal(t, "/skills/reel", parsed.Query().Get("path"), "the page on the neighbour, not the neighbour's address")

	user, err := hub.app.Queries.UserByEmail(ctx, "reader@example.com")
	require.NoError(t, err)
	b := newBrowser()
	b.cookies[hub.host] = hub.sessionFor(t, user.ID, "user")
	page := b.get(t, hub, parsed.RequestURI())
	require.Equal(t, http.StatusOK, page.status, page.body)
	_, form := b.pressOpenButton(t, hub, page)
	landed := b.arrive(t, peer, form, hub.url)
	require.Equal(t, http.StatusFound, landed.status, landed.body)
	require.Equal(t, "/skills/reel", landed.location)

	deeper := hubStore.tool(t, "federated_search", `{"kb_id":"peer/second","query":"zzfindmezz"}`)
	if deeper.Error == nil {
		var deeperPayload mcp.SearchResultPayload
		if raw, ok := deeper.Result.(mcp.CallToolResult).StructuredContent.(json.RawMessage); ok && json.Unmarshal(raw, &deeperPayload) == nil {
			for _, item := range deeperPayload.Results {
				require.Empty(t, item.OpenURL, "a result from a second hop gets no open link in this version")
			}
		}
	}
}

const leavePath = "/_system/federation/leave"

func (b *browser) guestPageBody(t *testing.T, s *site, path string) string {
	t.Helper()
	return b.get(t, s, path).body
}

func TestSecurityTheBarNamesTheHubAndAGuestCanLeave(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)

	for _, path := range []string{"/", "/shared", "/other"} {
		body := b.guestPageBody(t, s.peer, path)
		require.Contains(t, body, `class="guest-bar"`, path)
		require.Contains(t, body, "hub.test", "the bar names the hub: %s", path)
		require.Contains(t, body, `href="http://hub.test/"`, "Back leads to the hub's origin")
		require.Contains(t, body, `action="`+leavePath+`"`, "Leave is a form post")
	}

	local := newBrowser()
	local.cookies[s.peer.host] = s.peer.signIn(t, "local@example.com")
	for _, reader := range []*browser{newBrowser(), local} {
		require.NotContains(t, reader.guestPageBody(t, s.peer, "/"), "guest-bar", "only a guest sees the bar")
	}

	left := b.post(t, s.peer, leavePath, url.Values{}, s.peer.url)
	require.Equal(t, http.StatusSeeOther, left.status, left.body)
	require.Equal(t, "http://hub.test/", left.location, "leaving goes back to the hub")
	require.Empty(t, b.cookies[s.peer.host], "and ends the guest session")
	require.Equal(t, "200[front]", b.page(t, s.peer, "/"))
	require.Equal(t, "401[]", b.page(t, s.peer, "/shared"))
}

func TestSecurityLeavingTouchesNoOneButAGuest(t *testing.T) {
	s := seedTwoSites(t)

	own := newBrowser()
	own.cookies[s.peer.host] = s.peer.signIn(t, "local@example.com")
	before := own.cookies[s.peer.host]
	r := own.post(t, s.peer, leavePath, url.Values{}, s.peer.url)
	require.Equal(t, http.StatusForbidden, r.status, "a reader with their own session is not a guest, and keeps the session")
	require.Equal(t, before, own.cookies[s.peer.host])

	require.Equal(t, http.StatusForbidden, newBrowser().post(t, s.peer, leavePath, url.Values{}, s.peer.url).status)
}

func TestSecurityALayoutShowsTheBarOnlyWhenItCallsTheMethod(t *testing.T) {
	s := seedTwoSites(t)
	insertLatestNote(t, s.peer.app, "_layouts/withbar.html", "<html><body>{{ defaultTemplate.GuestBar() }}<main>zzlayoutzz</main></body></html>")
	insertLatestNote(t, s.peer.app, "_layouts/nobar.html", "<html><body><main>zzlayoutzz</main></body></html>")
	insertLatestNote(t, s.peer.app, "withbar.md", "---\nfree: true\nlayout: withbar\n---\nbody\n")
	insertLatestNote(t, s.peer.app, "nobar.md", "---\nfree: true\nlayout: nobar\n---\nbody\n")
	s.peer.load(t)

	guest := s.guest(t)
	local := newBrowser()
	local.cookies[s.peer.host] = s.peer.signIn(t, "local@example.com")
	admin := newBrowser()
	admin.cookies[s.peer.host] = s.peer.sessionFor(t, s.peer.adminID, "admin")

	shown := guest.get(t, s.peer, "/withbar")
	require.Equal(t, http.StatusOK, shown.status, shown.body)
	require.Contains(t, shown.body, "zzlayoutzz")
	require.Contains(t, shown.body, `class="guest-bar"`)
	require.Contains(t, shown.body, "hub.test")

	for name, reader := range map[string]*browser{"anonymous": newBrowser(), "local": local, "admin": admin} {
		body := reader.guestPageBody(t, s.peer, "/withbar")
		require.Contains(t, body, "zzlayoutzz", name)
		require.NotContains(t, body, "guest-bar", "%s: the method answers an empty string", name)
	}

	require.NotContains(t, guest.guestPageBody(t, s.peer, "/nobar"), "guest-bar", "a layout that does not call the method shows none")
}

func TestSecurityTheOriginInTheBarIsEscaped(t *testing.T) {
	s := seedTwoSites(t)
	key, err := s.peer.app.Queries.FederationSecretByKID(context.Background(), "hub-to-peer")
	require.NoError(t, err)

	claims := jwt.MapClaims{
		"i": s.peer.adminID + 100, "r": usertoken.RoleFederationGuest, "k": "hub-to-peer", "ki": key.ID, "h": 1, "rd": s.reader,
		"o":   `http://a"onmouseover="x<script>alert(1)</script>.test`,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.peer.app.config.UserToken.Secret))
	require.NoError(t, err)

	b := newBrowser()
	b.cookies[s.peer.host] = signed
	body := b.guestPageBody(t, s.peer, "/")
	require.NotContains(t, body, `"onmouseover="`)
	require.NotContains(t, body, "<script>alert(1)")
}

func TestSecurityAGuestWhoWasLetGoIsToldOnceHowToComeBack(t *testing.T) {
	s := seedTwoSites(t)
	b := s.guest(t)
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"))

	const line = "You were here as a guest"
	require.NoError(t, s.peer.app.WriteQueries.RevokeFederationSecret(context.Background(), s.edgeID))

	first := b.get(t, s.peer, "/shared")
	require.Equal(t, http.StatusUnauthorized, first.status)
	require.Contains(t, first.body, line, "the wall says why the guest is not let in")
	require.Contains(t, first.body, "Open this base again from your own knowledge base")

	second := b.get(t, s.peer, "/shared")
	require.NotContains(t, second.body, line, "once")

	other := newBrowser()
	require.NotContains(t, other.guestPageBody(t, s.peer, "/shared"), line, "a reader who never was a guest is not told")
}

func TestSecurityTheRefusalPageLinksBackOnlyForAValidlySignedToken(t *testing.T) {
	s := seedTwoSites(t)
	s.peer.acceptGuests(t, "hub-to-peer", false)

	signed := newBrowser().arrive(t, s.peer, handoffForm(handoffToken(t, "hub-to-peer", nil)), "http://attacker.test")
	require.Equal(t, http.StatusForbidden, signed.status)
	require.Contains(t, signed.body, `href="http://hub.test"`, "the token is genuine, so the page leads back to its hub")
	require.NotContains(t, signed.body, "attacker.test", "and never to the site that posted it")

	token := handoffToken(t, "hub-to-peer", nil)
	forged := newBrowser().arrive(t, s.peer, handoffForm(token[:len(token)-4]+"AAAA"), hubOrigin)
	require.Equal(t, http.StatusForbidden, forged.status)
	require.NotContains(t, forged.body, "hub.test", "a forged token names no hub")

	unknown := newBrowser().arrive(t, s.peer, handoffForm(handoffToken(t, "nobody", map[string]any{"org": "http://evil.test"})), "http://evil.test")
	require.Equal(t, http.StatusForbidden, unknown.status)
	require.NotContains(t, unknown.body, "evil.test")
}

func TestSecurityTheDocumentedPatchClosesNotesWithNoSubgraphToAGuest(t *testing.T) {
	s := seedTwoSites(t)
	ctx := context.Background()
	insertLatestNote(t, s.peer.app, "plain.md", "zzplainzz\n")
	s.peer.load(t)
	b := s.guest(t)

	before := b.get(t, s.peer, "/plain")
	require.Equal(t, http.StatusOK, before.status, "without the patch a guest reads a note with no subgraph, as an agent on the edge does")
	require.Contains(t, before.body, "zzplainzz")

	patch := db.InsertFrontmatterPatchParams{
		IncludePatterns: `["**"]`, ExcludePatterns: `[]`, Description: "Private by default", Enabled: true, CreatedBy: s.peer.adminID,
		Jsonnet: "if std.objectHas(meta, \"subgraph\") || std.objectHas(meta, \"subgraphs\")\nthen {}\nelse { subgraph: \"private\" }",
	}
	_, err := s.peer.app.WriteQueries.InsertFrontmatterPatch(ctx, patch)
	require.NoError(t, err)
	require.NoError(t, s.peer.app.WriteQueries.InsertSubgraph(ctx, "private"))
	s.peer.load(t)

	after := b.get(t, s.peer, "/plain")
	require.Equal(t, http.StatusForbidden, after.status, "with the patch the same guest is refused")
	require.NotContains(t, after.body, "zzplainzz")
	require.Equal(t, "200[shared]", b.page(t, s.peer, "/shared"), "and still reads what the edge grants")
}

func (b *browser) subscribe(t *testing.T, s *site, query string) reply {
	t.Helper()

	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)

	fctx := &fasthttp.RequestCtx{}
	fctx.Init(&fasthttp.Request{}, nil, nil)
	fctx.Request.Header.SetMethod(http.MethodPost)
	fctx.Request.SetRequestURI("/_system/graphql")
	fctx.Request.Header.SetHost(s.host)
	fctx.Request.Header.SetContentType("application/json")
	fctx.Request.Header.Set("Accept", "text/event-stream")
	fctx.Request.SetBody(body)
	fctx.Request.Header.SetCookie("trip2g_token", b.cookies[s.host])

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handler(fctx)
		if fctx.Response.IsBodyStream() {
			_, _ = fctx.Response.WriteTo(&out)
		} else {
			out.Write(fctx.Response.Body())
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the subscription stayed open")
	}

	return reply{status: fctx.Response.StatusCode(), body: out.String()}
}

func TestSecurityAGuestCannotOpenASubscription(t *testing.T) {
	s := seedTwoSites(t)
	insertLatestNote(t, s.peer.app, "open.md", "---\nfree: true\n---\nzzopenzz\n")
	s.peer.load(t)
	b := s.guest(t)

	r := b.subscribe(t, s.peer, `subscription { noteChanges(filter: {includePatterns: ["**"]}) { changes { __typename } } }`)
	require.Contains(t, r.body, "a federation guest can only read", "status %d: %s", r.status, r.body)
	require.NotContains(t, r.body, "open.md")
}
