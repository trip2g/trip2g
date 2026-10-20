package mdloader_test

import (
	"encoding/json"
	"os"
	"sync"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdloader"

	"github.com/stretchr/testify/require"
)

const domainGoldenFile = "testdata/domain_html_golden.json"

func domainGoldenHosts() []string {
	return []string{"", "foo.com", "bar.com", "unknown.com"}
}

func domainGoldenSources() []mdloader.SourceFile {
	files := map[string]string{
		"foo_home.md": "---\nfree: true\nroute: foo.com/\nheader: nav.md\n---\n# Foo\n\n[[plain]] [[bar_page]] [[shared#Second]]\n\n![[embedded]]",
		"bar_page.md": "---\nfree: true\nroutes:\n  - bar.com/page\n  - /bar-alias\n---\n# Bar\n\n[[plain]] [[foo_home]]",
		"nav.md":      "---\nfree: true\n---\n- [[foo_home]]\n- [[bar_page]]\n- [[plain]]\n- [[paid]]",
		"plain.md":    "---\nfree: true\n---\n# Plain\n\nBack to [[foo_home]], over to [[bar_page|Bar]], see [[shared]] and [[missing]].\n\n![[embedded]]\n\n![[pic.png]]",
		"shared.md":   "---\nfree: true\n---\n# Shared\n\n## Second\n\n[[plain]] and [[paid]] and [[#Second]]",
		"embedded.md": "---\nfree: true\n---\nEmbedded links to [[bar_page]].",
		"paid.md":     "---\nsubgraph: x\n---\n# Paid\n\n[[foo_home]]",
		"isolated.md": "---\nfree: true\n---\nNo links here.",
		"core.md":     "# Core\n\n[[plain]]",
	}
	srcs := make([]mdloader.SourceFile, 0, len(files))
	id := int64(0)
	for _, path := range []string{"foo_home.md", "bar_page.md", "nav.md", "plain.md", "shared.md", "embedded.md", "paid.md", "isolated.md", "core.md"} {
		id++
		srcs = append(srcs, mdloader.SourceFile{Path: path, PathID: id, Content: []byte(files[path])})
	}
	return srcs
}

func loadDomainGolden(t *testing.T) map[string]map[string]string {
	t.Helper()
	raw, err := os.ReadFile(domainGoldenFile)
	require.NoError(t, err)
	var golden map[string]map[string]string
	require.NoError(t, json.Unmarshal(raw, &golden))
	return golden
}

// TestLazyDomainHTMLMatchesEager compares what every note serves on every host
// with the golden file, which the eager loader (every note pre-rendered for
// every custom host) produced for the same vault. Notes reached on a host
// only through links, embeds and headers -- plain, shared, nav, embedded --
// are the ones rendered lazily now.
func TestLazyDomainHTMLMatchesEager(t *testing.T) {
	golden := loadDomainGolden(t)
	nvs, err := mdloader.Load(mdloader.Options{Sources: domainGoldenSources(), Log: &logger.TestLogger{}, PublicURL: "https://main.test"})
	require.NoError(t, err)
	require.Len(t, nvs.PathMap, len(golden))

	for path, hosts := range golden {
		nv := nvs.PathMap[path]
		require.NotNil(t, nv, path)
		for host, want := range hosts {
			require.Equal(t, want, string(nv.HTMLForHost(host)), "%s on %q", path, host)
		}
	}
}

// TestOwnRouteHostIsPreRendered checks that load still renders a note for the
// hosts it has routes on, and leaves the others' notes to request time.
func TestOwnRouteHostIsPreRendered(t *testing.T) {
	golden := loadDomainGolden(t)
	nvs, err := mdloader.Load(mdloader.Options{Sources: domainGoldenSources(), Log: &logger.TestLogger{}, PublicURL: "https://main.test"})
	require.NoError(t, err)

	home := nvs.PathMap["foo_home.md"]
	require.Contains(t, home.DomainHTML, "foo.com")
	require.Equal(t, golden["foo_home.md"]["foo.com"], string(home.DomainHTML["foo.com"]))
	require.NotContains(t, home.DomainHTML, "bar.com")

	bar := nvs.PathMap["bar_page.md"]
	require.Contains(t, bar.DomainHTML, "bar.com")
	require.Equal(t, golden["bar_page.md"]["bar.com"], string(bar.DomainHTML["bar.com"]))

	plain := nvs.PathMap["plain.md"]
	require.Contains(t, plain.DomainHTML, "", "main-domain pass stays eager")
	require.NotContains(t, plain.DomainHTML, "foo.com")
	require.NotContains(t, plain.DomainHTML, "bar.com")
}

// TestLazyDomainHTMLConcurrent asks for the same and different (note, host)
// pairs from many goroutines, through snapshot copies, next to partial
// renders that share the renderer. Run with -race.
func TestLazyDomainHTMLConcurrent(t *testing.T) {
	golden := loadDomainGolden(t)
	nvs, err := mdloader.Load(mdloader.Options{Sources: domainGoldenSources(), Log: &logger.TestLogger{}, PublicURL: "https://main.test"})
	require.NoError(t, err)

	paths := []string{"plain.md", "shared.md", "nav.md", "embedded.md", "foo_home.md"}

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snapshot := nvs.Copy()
			for j := range 20 {
				path := paths[(i+j)%len(paths)]
				for _, host := range domainGoldenHosts() {
					nv := snapshot.PathMap[path]
					if string(nv.HTMLForHost(host)) != golden[path][host] {
						t.Errorf("%s on %q differs from eager", path, host)
					}
					nv.PartialRenderer.Introduce()
					nv.PartialRenderer.Sections(2)
				}
			}
		}()
	}
	wg.Wait()
}
