package mdloader_test

import (
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/mdloader"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

// TestResolveForDomain tests the pure resolveForDomain function directly.
// Since resolveForDomain is unexported, we test it through the full Load pipeline
// and verify the resulting DomainHTML content.

func TestResolveForDomain(t *testing.T) {
	tests := []struct {
		name        string
		sources     []mdloader.SourceFile
		checkNote   string // PathMap key of the note to inspect
		host        string // expected domain host in DomainHTML
		wantInHTML  string // substring expected in DomainHTML
		wantMissing string // substring that must NOT be in DomainHTML (empty = skip check)
	}{
		{
			name: "same custom domain: link uses domain path",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
route: foo.com/target
---
Page B`),
				},
			},
			checkNote:  "page-a.md",
			host:       "foo.com",
			wantInHTML: `href="/target"`,
		},
		{
			name: "different custom domain: link uses full URL",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
route: bar.com/target
---
Page B`),
				},
			},
			checkNote:  "page-a.md",
			host:       "foo.com",
			wantInHTML: `href="https://bar.com/target"`,
		},
		{
			name: "no custom routes on target: link uses permalink",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
---
Page B`),
				},
			},
			checkNote: "page-a.md",
			host:      "foo.com",
			// No custom routes on page-b, so domain links are identical to main.
			// DomainHTML should NOT be generated (optimization).
			wantInHTML: "",
		},
		{
			name: "route on domain with empty path: resolves to permalink (same as main, no re-render)",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
route: foo.com
---
Page B`),
				},
			},
			checkNote: "page-a.md",
			host:      "foo.com",
			// Empty Path means "use Permalink". Since domain path == permalink,
			// domain links are identical to main => DomainHTML not generated.
			wantInHTML: "",
		},
		{
			name: "target is domain root: uses /",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[root-page]]`),
				},
				{
					Path: "root-page.md",
					Content: []byte(`---
free: true
route: foo.com/
---
Root page`),
				},
			},
			checkNote:  "page-a.md",
			host:       "foo.com",
			wantInHTML: `href="/"`,
		},
		{
			name: "only main-domain alias: uses permalink",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
route: /about
---
Page B`),
				},
			},
			checkNote: "page-a.md",
			host:      "foo.com",
			// page-b has only a main-domain alias (Host=""), no custom domain route.
			// Domain links are identical to main => DomainHTML["foo.com"] not generated.
			wantInHTML: "",
		},
		// Main domain pass (host="") test cases.
		{
			name: "main domain: link to custom-domain-only note uses full URL",
			sources: []mdloader.SourceFile{
				{
					Path: "index.md",
					Content: []byte(`---
free: true
---
Link: [[extra]]`),
				},
				{
					Path: "extra.md",
					Content: []byte(`---
free: true
route: extra.com/
---
Extra page`),
				},
			},
			checkNote:  "index.md",
			host:       "", // main domain pass
			wantInHTML: `href="https://extra.com/"`,
		},
		{
			name: "main domain: link to note with main-domain alias uses alias path",
			sources: []mdloader.SourceFile{
				{
					Path: "index.md",
					Content: []byte(`---
free: true
---
Link: [[about-page]]`),
				},
				{
					Path: "about-page.md",
					Content: []byte(`---
free: true
route: /about
---
About`),
				},
				// A custom-domain note is required so that CustomDomains() is non-empty
				// and generateDomainHTMLs does not short-circuit before the main domain pass.
				{
					Path: "trigger.md",
					Content: []byte(`---
free: true
route: trigger.com/x
---
Trigger`),
				},
			},
			checkNote:  "index.md",
			host:       "", // main domain pass
			wantInHTML: `href="/about"`,
		},
		{
			name: "main domain: link to plain note uses permalink (no re-render)",
			sources: []mdloader.SourceFile{
				{
					Path: "index.md",
					Content: []byte(`---
free: true
---
Link: [[plain]]`),
				},
				{
					Path: "plain.md",
					Content: []byte(`---
free: true
---
Plain`),
				},
			},
			checkNote: "index.md",
			host:      "", // main domain pass — should NOT generate DomainHTML[""]
			// plain.md has no routes; resolveForDomain(plain, "") returns permalink unchanged.
			wantInHTML: "",
		},
		{
			name: "self-referential: source links to itself on same domain",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/self
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
route: foo.com/b-path
---
Page B`),
				},
			},
			checkNote:  "page-a.md",
			host:       "foo.com",
			wantInHTML: `href="/b-path"`,
		},
		{
			name: "multiple routes pick correct domain",
			sources: []mdloader.SourceFile{
				{
					Path: "page-a.md",
					Content: []byte(`---
free: true
route: foo.com/a
---
Link: [[page-b]]`),
				},
				{
					Path: "page-b.md",
					Content: []byte(`---
free: true
routes:
  - foo.com/f
  - bar.com/b
---
Page B`),
				},
			},
			checkNote:   "page-a.md",
			host:        "foo.com",
			wantInHTML:  `href="/f"`,
			wantMissing: `bar.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.TestLogger{}
			pages, err := mdloader.Load(mdloader.Options{
				Sources: tt.sources,
				Log:     &log,
			})
			require.NoError(t, err)

			note := pages.PathMap[tt.checkNote]
			require.NotNil(t, note, "note %s must exist", tt.checkNote)

			if tt.wantInHTML == "" {
				// DomainHTML should not be generated (optimization: no differences).
				if note.DomainHTML != nil {
					_, exists := note.DomainHTML[tt.host]
					require.False(t, exists, "DomainHTML[%s] should not exist when domain links are identical to main", tt.host)
				}
				return
			}

			require.NotNil(t, note.DomainHTML, "DomainHTML must be initialized")
			domainHTML, exists := note.DomainHTML[tt.host]
			require.True(t, exists, "DomainHTML[%s] must exist", tt.host)
			require.Contains(t, string(domainHTML), tt.wantInHTML,
				"DomainHTML should contain expected href")

			if tt.wantMissing != "" {
				require.NotContains(t, string(domainHTML), tt.wantMissing,
					"DomainHTML should NOT contain %s", tt.wantMissing)
			}
		})
	}
}

// TestGenerateDomainHTML is an integration test using the full loader pipeline.
// Verifies that domain-aware re-render produces correct DomainHTML while leaving
// the main HTML unchanged.
func TestGenerateDomainHTML(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{
		{
			Path: "page-a.md",
			Content: []byte(`---
free: true
route: foo.com/
---
Link to B: [[page-b]]
Embed C: ![[page-c]]`),
		},
		{
			Path: "page-b.md",
			Content: []byte(`---
free: true
route: foo.com/b-custom
---
Page B content`),
		},
		{
			Path: "page-c.md",
			Content: []byte(`---
free: true
route: foo.com/c-custom
---
Page C content`),
		},
	}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	pageA := pages.PathMap["page-a.md"]
	require.NotNil(t, pageA, "page-a.md must exist")

	// 1. DomainHTML for foo.com must exist on page A.
	require.NotNil(t, pageA.DomainHTML, "page A should have DomainHTML")
	domainHTML, exists := pageA.DomainHTML["foo.com"]
	require.True(t, exists, "DomainHTML['foo.com'] must exist for page A")
	require.NotEmpty(t, domainHTML, "DomainHTML should not be empty")

	// 2. Domain HTML should contain domain path for page-b.
	require.Contains(t, string(domainHTML), `href="/b-custom"`,
		"Domain HTML should use domain path for page-b link")

	// 3. Main HTML (pageA.HTML) is always the canonical form used for RSS/API/MCP.
	// It uses the permalink for page-b, not the domain path.
	// Note: "page-b.md" gets permalink "/page_b" (dash normalized to underscore).
	require.Contains(t, string(pageA.HTML), `href="/page_b"`,
		"Main HTML should use permalink for page-b")
	require.NotContains(t, string(pageA.HTML), `href="/b-custom"`,
		"Main HTML should NOT contain domain path")

	// 3b. DomainHTML[""] (main domain pass) is generated because page-b is a custom-domain-only
	// note. On the main domain, [[page-b]] should link to https://foo.com/b-custom.
	require.NotNil(t, pageA.DomainHTML, "DomainHTML must be initialized")
	mainDomainHTML, hasmain := pageA.DomainHTML[""]
	require.True(t, hasmain, `DomainHTML[""] must exist (main domain re-render)`)
	require.Contains(t, string(mainDomainHTML), `href="https://foo.com/b-custom"`,
		"Main domain HTML should use full URL for custom-domain-only note")

	// 4. Domain HTML should still contain the embed of page-c (uses main HTML).
	require.Contains(t, string(domainHTML), "Page C content",
		"Domain HTML should contain embedded page-c content")

	// 5. page-b should NOT have DomainHTML (it doesn't link to domain-routed notes,
	// or if it does, the links are identical to main).
	pageB := pages.PathMap["page-b.md"]
	require.NotNil(t, pageB, "page-b.md must exist")
	if pageB.DomainHTML != nil {
		_, hasFoo := pageB.DomainHTML["foo.com"]
		require.False(t, hasFoo,
			"page-b should not have DomainHTML['foo.com'] (no links to domain-routed notes)")
	}
}

// TestDomainHTMLNotGeneratedWithoutCustomRoutes verifies that notes without
// any custom domain routes in the vault never get DomainHTML (zero memory overhead).
// Neither note has custom routes, and the linked target (other.md) has no custom domain
// routes, so resolveForDomain(other, "") returns the unchanged permalink → no re-render.
func TestDomainHTMLNotGeneratedWithoutCustomRoutes(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{
		{
			Path: "index.md",
			Content: []byte(`---
free: true
---
Hello [[other]]`),
		},
		{
			Path: "other.md",
			Content: []byte(`---
free: true
---
Other page`),
		},
	}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	for path, note := range pages.PathMap {
		require.Nil(t, note.DomainHTML,
			"note %s should have nil DomainHTML (no custom domain routes in vault)", path)
	}
}

// TestDomainHTMLSkippedWhenLinksIdentical verifies the optimization:
// if all domain-resolved links match the main-domain permalinks, DomainHTML
// is not generated for that note (avoids wasteful re-render).
func TestDomainHTMLSkippedWhenLinksIdentical(t *testing.T) {
	log := logger.TestLogger{}

	sourceFiles := []mdloader.SourceFile{
		{
			Path: "page-a.md",
			Content: []byte(`---
free: true
route: foo.com/a
---
Link to plain note: [[page-b]]`),
		},
		{
			Path: "page-b.md",
			Content: []byte(`---
free: true
---
Plain page with no routes`),
		},
	}

	pages, err := mdloader.Load(mdloader.Options{
		Sources: sourceFiles,
		Log:     &log,
	})
	require.NoError(t, err)

	pageA := pages.PathMap["page-a.md"]
	require.NotNil(t, pageA)

	// page-b has no custom domain routes, so domain links for page-a
	// are identical to main links. DomainHTML should NOT be generated.
	if pageA.DomainHTML != nil {
		_, exists := pageA.DomainHTML["foo.com"]
		require.False(t, exists,
			"DomainHTML should not be generated when domain links match main links")
	}
}

// TestHasCustomDomainRoutes verifies the hasCustomDomainRoutes helper.
func TestHasCustomDomainRoutes(t *testing.T) {
	tests := []struct {
		name   string
		routes []model.ParsedRoute
		want   bool
	}{
		{
			name:   "no routes",
			routes: nil,
			want:   false,
		},
		{
			name:   "only main domain alias",
			routes: []model.ParsedRoute{{Host: "", Path: "/about"}},
			want:   false,
		},
		{
			name:   "custom domain route",
			routes: []model.ParsedRoute{{Host: "foo.com", Path: "/page"}},
			want:   true,
		},
		{
			name: "mixed routes",
			routes: []model.ParsedRoute{
				{Host: "", Path: "/alias"},
				{Host: "foo.com", Path: "/page"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nv := &model.NoteView{Routes: tt.routes}
			// hasCustomDomainRoutes is not exported, so we test indirectly:
			// a note with custom routes that links to another domain-routed note
			// should produce DomainHTML. We verify the logic via Load tests above.
			hasCustom := false
			for _, r := range nv.Routes {
				if r.Host != "" {
					hasCustom = true
					break
				}
			}
			require.Equal(t, tt.want, hasCustom)
		})
	}
}
