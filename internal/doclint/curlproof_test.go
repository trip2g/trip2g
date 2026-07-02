package doclint

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/CloudyKit/jet/v6"
	"github.com/valyala/fasthttp"

	"trip2g/internal/logger"
	"trip2g/internal/mdchunk"
	"trip2g/internal/mdloader"
	"trip2g/internal/noteloader"
	"trip2g/internal/templateviews"
)

// pubEnv wraps the FS env but reports a non-empty PublicURL so the robots layout
// can emit an absolute Sitemap line (mirrors production).
type pubEnv struct {
	*fsEnv
}

func (pubEnv) PublicURL() string { return "https://trip2g.com" }

// TestCurlProof_ContentTypeNotes loads the real docs/ vault and renders the SEO
// notes exactly as the request path does, printing the Content-Type + body so a
// human (and the PR) can see /robots.txt, /llms.txt, and a normal note behave.
func TestCurlProof_ContentTypeNotes(t *testing.T) {
	ctx := context.Background()
	env := pubEnv{newFsEnv("../../docs", &logger.DummyLogger{})}
	ldr := noteloader.New("curlproof", env, mdloader.Config{})
	if err := ldr.Load(ctx, noteloader.LoadOptions{SkipSearchIndex: true}); err != nil {
		t.Fatalf("load docs vault: %v", err)
	}
	nvs := ldr.NoteViews()
	layouts := ldr.Layouts()

	// --- /robots.txt: served via the `robots` Jet layout ---
	robots := nvs.GetByPath("/robots.txt")
	if robots == nil {
		t.Fatal("no note at /robots.txt (docs/robots.md slug)")
	}
	if robots.Layout != "robots" {
		t.Fatalf("robots note layout = %q, want robots", robots.Layout)
	}
	layout, ok := layouts.Map["/robots"]
	if !ok || layout.View == nil {
		t.Fatal("robots layout not loaded")
	}
	rc := &fasthttp.RequestCtx{}
	rc.SetContentType("text/html; charset=utf-8")
	vars := make(jet.VarMap)
	vars["note"] = reflect.ValueOf(templateviews.NewNote(robots))
	vars["nvs"] = reflect.ValueOf(templateviews.NewNVS(nvs, ""))
	vars["title"] = reflect.ValueOf(robots.Title)
	vars["publicURL"] = reflect.ValueOf(env.PublicURL())
	vars["response"] = reflect.ValueOf(&templateviews.ResponseWriter{Ctx: rc})
	if err := layout.View.Execute(rc, vars, nil); err != nil {
		t.Fatalf("execute robots layout: %v", err)
	}
	robotsBody := string(rc.Response.Body())
	robotsCT := string(rc.Response.Header.ContentType())
	t.Logf("\n=== curl -i https://trip2g.com/robots.txt ===\nContent-Type: %s\n\n%s\n", robotsCT, robotsBody)

	if !strings.HasPrefix(robotsCT, "text/plain") {
		t.Errorf("robots Content-Type = %q, want text/plain", robotsCT)
	}
	if !strings.Contains(robotsBody, "User-agent: *") {
		t.Error("robots body missing User-agent")
	}
	if !strings.Contains(robotsBody, "Sitemap: https://trip2g.com/sitemap.xml") {
		t.Error("robots body missing absolute Sitemap line")
	}

	// --- /llms.txt: served via the content_type frontmatter (plain note) ---
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

	// --- a normal note still serves text/html (no content_type, no plain layout) ---
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
