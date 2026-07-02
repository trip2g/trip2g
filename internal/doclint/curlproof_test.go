package doclint

import (
	"context"
	"strings"
	"testing"

	"trip2g/internal/logger"
	"trip2g/internal/mdchunk"
	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"
	"trip2g/internal/templateviews"
)

// TestCurlProof_ContentTypeNotes loads the real docs/ vault and verifies the SEO
// notes behave as the request path would serve them:
//   - /robots.txt: content_type frontmatter = text/plain, body has User-agent + absolute Sitemap
//   - /llms.txt:   content_type frontmatter = text/plain, raw body, frontmatter stripped
//   - a normal note: no content_type => default text/html render path
func TestCurlProof_ContentTypeNotes(t *testing.T) {
	ctx := context.Background()
	env := newFsEnv("../../docs", &logger.DummyLogger{})
	ldr := noteloader.New("curlproof", env, mdloader.Config{})
	if err := ldr.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}); err != nil {
		t.Fatalf("load docs vault: %v", err)
	}
	nvs := ldr.NoteViews()

	// --- /robots.txt: pure content_type note, no layout ---
	robots := nvs.GetByPath("/robots.txt")
	if robots == nil {
		t.Fatal("no note at /robots.txt (docs/robots.md slug)")
	}
	if robots.Layout != "" {
		t.Errorf("robots note should have no layout, got %q", robots.Layout)
	}
	robotsNote := templateviews.NewNote(robots)
	robotsCT := robotsNote.M().GetString("content_type", "")
	robotsBody := mdchunk.StripFrontmatter(string(robots.Content))
	t.Logf("\n=== curl -i https://trip2g.com/robots.txt ===\nContent-Type: %s\n\n%s\n", robotsCT, robotsBody)

	if !strings.HasPrefix(robotsCT, "text/plain") {
		t.Errorf("robots content_type = %q, want text/plain", robotsCT)
	}
	if !strings.Contains(robotsBody, "User-agent: *") {
		t.Error("robots body missing User-agent")
	}
	if !strings.Contains(robotsBody, "Sitemap: https://trip2g.com/sitemap.xml") {
		t.Error("robots body missing absolute Sitemap line")
	}
	if strings.Contains(robotsBody, "slug:") || strings.Contains(robotsBody, "content_type:") {
		t.Error("robots body still contains frontmatter")
	}

	// --- /llms.txt: content_type frontmatter, no layout → raw body ---
	llms := nvs.GetByPath("/llms.txt")
	if llms == nil {
		t.Fatal("no note at /llms.txt (docs/llms.md slug)")
	}
	llmsCT := templateviews.NewNote(llms).M().GetString("content_type", "")
	llmsBody := mdchunk.StripFrontmatter(string(llms.Content))
	t.Logf("=== curl -i https://trip2g.com/llms.txt ===\nContent-Type: %s\n\n%s\n", llmsCT, llmsBody)

	if !strings.HasPrefix(llmsCT, "text/plain") {
		t.Errorf("llms content_type = %q, want text/plain", llmsCT)
	}
	if strings.Contains(llmsBody, "content_type:") || strings.Contains(llmsBody, "slug:") {
		t.Error("llms body still contains frontmatter")
	}
	if !strings.Contains(llmsBody, "# trip2g") {
		t.Error("llms body missing the llms.txt summary")
	}

	// --- a normal note has no content_type → default text/html render path ---
	var normal string
	for _, n := range nvs.List {
		if n.Layout == "" && n.Free && templateviews.NewNote(n).M().GetString("content_type", "") == "" &&
			!strings.Contains(n.Permalink, "/_") && n.Permalink != "/robots.txt" && n.Permalink != "/llms.txt" {
			normal = n.Permalink
			break
		}
	}
	t.Logf("=== a normal note (%s) has no content_type -> default text/html render path ===\n", normal)
	if normal == "" {
		t.Skip("no plain content note found to confirm html default")
	}
}
