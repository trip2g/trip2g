package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"trip2g/internal/case/runnotequery"
	"trip2g/internal/dataencryption"
	"trip2g/internal/db"
	"trip2g/internal/noteloader"

	"github.com/kr/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestFederationGuestAccessMatrix runs guests through the read paths of the
// free-note matrix: a guest of an edge granting "paid", one granting only
// "elsewhere" (no note is in it), and one whose edge grants nothing.
func TestFederationGuestAccessMatrix(t *testing.T) {
	// Building a handler registers process-wide metrics, and the free-note
	// matrix builds one too.
	registerer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry() //nolint:reassign // restored below
	m := seedFreeNoteMatrix(t)
	prometheus.DefaultRegisterer = registerer //nolint:reassign // the original registry
	ctx := context.Background()
	a := m.app
	a.config.FederationAcceptGuests = true
	a.config.MCPFederationMaxDepth = 3
	encryption, err := dataencryption.NewManager(dataencryption.DefaultConfig())
	require.NoError(t, err)
	a.Manager = encryption
	a.NotesCache = runnotequery.NewNotesCache()

	insertLatestNote(t, a, "queries/all.md", "---\ntype: note-query\nfree: true\n---\n```jsonnet\n[n.path for n in notes]\n```\n")
	require.NoError(t, a.latestNoteLoader.Load(ctx, noteloader.LoadOptions{}))
	a.ClearPageCache()

	adminID := m.readers[len(m.readers)-1].userID
	peer := &site{store: &store{app: a, handler: m.handler, adminID: adminID}, url: "https://example.com", host: "example.com"}
	edges := map[string][]string{"fg-paid": {"paid"}, "fg-elsewhere": {"elsewhere"}, "fg-none": nil}
	names := make([]string, 0, len(edges))
	for kid, grants := range edges {
		crypt, cryptErr := a.EncryptData(edgeSecret(kid))
		require.NoError(t, cryptErr)
		secret := db.InsertFederationSecretParams{Kid: kid, SecretCrypt: crypt, CreatedBy: adminID}
		_, insertErr := a.WriteQueries.InsertFederationSecret(ctx, secret)
		require.NoError(t, insertErr)
		for _, name := range grants {
			sg, sgErr := a.Queries.SubgraphByName(ctx, name)
			require.NoError(t, sgErr)
			grant := db.InsertFederationSecretSubgraphParams{Kid: kid, SubgraphID: sg.ID, CreatedBy: adminID}
			require.NoError(t, a.WriteQueries.InsertFederationSecretSubgraph(ctx, grant))
		}

		b := newBrowser()
		token := handoffToken(t, kid, map[string]any{"sub": readerHash(kid + "@hub.test")})
		r := b.arrive(t, peer, handoffForm(token))
		require.Equal(t, http.StatusFound, r.status, r.body)
		m.cookies[kid] = b.cookies[peer.host]
		names = append(names, kid)
	}
	sort.Strings(names)

	t.Run("every read path", func(t *testing.T) {
		got := m.guestTable(t, names)
		t.Log("\n" + got)
		want := strings.TrimSpace(federationGuestAccessMatrix)
		require.Equal(t, want, got, pretty.Diff(strings.Split(want, "\n"), strings.Split(got, "\n")))
	})

	t.Run("unlabelled and require_signin notes", func(t *testing.T) {
		// Pinned, not endorsed: ResolveWithSubgraphs grants a note with no
		// subgraph and a note in a require_signin subgraph to any edge with at
		// least one subgraph, whichever it is. An edge with none gets free notes.
		n2, n6 := m.notes[1], m.notes[5]
		require.Equal(t, "signin-closed.md", n2.path)
		require.Equal(t, "open-closed.md", n6.path)
		require.Equal(t, "200[n2]", m.pageCell(t, "fg-elsewhere", n2))
		require.Equal(t, "200[n6]", m.pageCell(t, "fg-elsewhere", n6))
		require.Equal(t, "403:paywall[]", m.pageCell(t, "fg-none", n2))
		require.Equal(t, "403:paywall[]", m.pageCell(t, "fg-none", n6))
	})

	t.Run("page cache never crosses readers", func(t *testing.T) {
		for _, name := range names {
			m.readers = append(m.readers, matrixReader{name: name})
		}
		m.requirePageCacheKeepsReadersApart(t)
	})
}

const guestNoteQuery = `{ noteQuery(input: {query: "queries/all.md"}) {
	... on NoteQueryPayload { result }
	... on ErrorPayload { message }
} }`

func (m *freeNoteMatrix) guestTable(t *testing.T, readers []string) string {
	t.Helper()

	var lines []string
	for _, reader := range readers {
		searchResults := m.search(t, reader)
		rssItems := m.rss(t, reader)
		similar := m.similar(t, reader)
		queried := m.graphql(t, reader, guestNoteQuery, nil)
		require.Contains(t, queried, `"result":[`, queried)
		mcpReply := m.do(t, reader, http.MethodPost, "/_system/mcp", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`),
			map[string]string{"Content-Type": "application/json"})

		for _, note := range m.notes {
			similarCell, ok := similar[note.permalink]
			if !ok {
				similarCell = "-"
			}
			line := fmt.Sprintf(
				"%-4s %-12s page=%s gql=%s search=%s query=%s mcp=%d asset=%s cache=%s rss=%s similar=%s",
				note.key,
				reader,
				m.pageCell(t, reader, note),
				m.graphqlNoteCell(t, reader, note),
				m.searchCell(searchResults, note),
				yesNo(strings.Contains(queried, `"`+note.path+`"`)),
				mcpReply.status,
				m.assetCell(t, reader, note),
				m.cacheCell(t, reader, note),
				m.rssCell(rssItems, note),
				similarCell,
			)
			lines = append(lines, line)
		}
	}

	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

const federationGuestAccessMatrix = `
n1   fg-elsewhere page=200[n1] gql=[n1] search=[n1] query=+ mcp=403 asset=200 cache=- rss=[n1] similar=[n1]
n1   fg-none      page=200[n1] gql=[n1] search=[n1] query=+ mcp=403 asset=200 cache=- rss=[n1] similar=[n1]
n1   fg-paid      page=200[n1] gql=[n1] search=[n1] query=+ mcp=403 asset=200 cache=- rss=[n1] similar=[n1]
n10  fg-elsewhere page=200[n10] gql=[n10] search=[n10] query=+ mcp=403 asset=200 cache=- rss=- similar=[n10]
n10  fg-none      page=200[n10] gql=[n10] search=[n10] query=+ mcp=403 asset=200 cache=- rss=- similar=[n10]
n10  fg-paid      page=200[n10] gql=[n10] search=[n10] query=+ mcp=403 asset=200 cache=- rss=- similar=[n10]
n2   fg-elsewhere page=200[n2] gql=[n2] search=[n2] query=+ mcp=403 asset=200 cache=- rss=- similar=[n2]
n2   fg-none      page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n2   fg-paid      page=200[n2] gql=[n2] search=[n2] query=+ mcp=403 asset=200 cache=- rss=- similar=[n2]
n3   fg-elsewhere page=200[n3] gql=[n3] search=[n3] query=+ mcp=403 asset=200 cache=- rss=[n3] similar=[n3]
n3   fg-none      page=200[n3] gql=[n3] search=[n3] query=+ mcp=403 asset=200 cache=- rss=[n3] similar=[n3]
n3   fg-paid      page=200[n3] gql=[n3] search=[n3] query=+ mcp=403 asset=200 cache=- rss=[n3] similar=[n3]
n4   fg-elsewhere page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n4   fg-none      page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n4   fg-paid      page=200[n4] gql=[n4] search=[n4] query=+ mcp=403 asset=200 cache=- rss=- similar=[n4]
n5   fg-elsewhere page=200[n5] gql=[n5] search=[n5] query=+ mcp=403 asset=200 cache=- rss=[n5] similar=[n5]
n5   fg-none      page=200[n5] gql=[n5] search=[n5] query=+ mcp=403 asset=200 cache=- rss=[n5] similar=[n5]
n5   fg-paid      page=200[n5] gql=[n5] search=[n5] query=+ mcp=403 asset=200 cache=- rss=[n5] similar=[n5]
n6   fg-elsewhere page=200[n6] gql=[n6] search=[n6] query=+ mcp=403 asset=200 cache=- rss=- similar=[n6]
n6   fg-none      page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n6   fg-paid      page=200[n6] gql=[n6] search=[n6] query=+ mcp=403 asset=200 cache=- rss=- similar=[n6]
n7   fg-elsewhere page=404[] gql=error search=- query=- mcp=403 asset=403 cache=- rss=- similar=-
n7   fg-none      page=404[] gql=error search=- query=- mcp=403 asset=403 cache=- rss=- similar=-
n7   fg-paid      page=404[] gql=error search=- query=- mcp=403 asset=403 cache=- rss=- similar=-
n8   fg-elsewhere page=404[] gql=error search=- query=- mcp=403 asset=- cache=- rss=- similar=-
n8   fg-none      page=404[] gql=error search=- query=- mcp=403 asset=- cache=- rss=- similar=-
n8   fg-paid      page=404[] gql=error search=- query=- mcp=403 asset=- cache=- rss=- similar=-
n9a  fg-elsewhere page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n9a  fg-none      page=403:paywall[] gql=error search=[] query=- mcp=403 asset=403 cache=- rss=- similar=-
n9a  fg-paid      page=200[n3,n9a] gql=[n3,n9a] search=[n3,n9a] query=+ mcp=403 asset=200 cache=- rss=- similar=[n3,n9a]
n9b  fg-elsewhere page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] query=+ mcp=403 asset=200 cache=- rss=[n4,n9b] similar=[n4,n9b]
n9b  fg-none      page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] query=+ mcp=403 asset=200 cache=- rss=[n4,n9b] similar=[n4,n9b]
n9b  fg-paid      page=200[n4,n9b] gql=[n4,n9b] search=[n4,n9b] query=+ mcp=403 asset=200 cache=- rss=[n4,n9b] similar=[n4,n9b]
n9c  fg-elsewhere page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] query=+ mcp=403 asset=200 cache=- rss=[n2,n9c] similar=[n2,n9c]
n9c  fg-none      page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] query=+ mcp=403 asset=200 cache=- rss=[n2,n9c] similar=[n2,n9c]
n9c  fg-paid      page=200[n2,n9c] gql=[n2,n9c] search=[n2,n9c] query=+ mcp=403 asset=200 cache=- rss=[n2,n9c] similar=[n2,n9c]
`
