package model

import (
	"html/template"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingDomainRenderer struct {
	renders atomic.Int64
}

func (r *countingDomainRenderer) RendersHost(host string) bool {
	return host == "foo.com" || host == "bar.com"
}

func (r *countingDomainRenderer) RenderDomainHTML(note *NoteView, host string) (template.HTML, bool) {
	r.renders.Add(1)
	if host == "bar.com" {
		return "", false
	}
	return template.HTML(note.Path + "@" + host), true
}

func TestHTMLForHostRendersOncePerSnapshot(t *testing.T) {
	renderer := &countingDomainRenderer{}
	nvs := NewNoteViews()
	note := &NoteView{Path: "a.md", Permalink: "/a", HTML: "main", DomainHTML: map[string]template.HTML{"": "main-domain"}}
	note.SetDomainHTMLRenderer(renderer)
	nvs.RegisterNote(note)

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			snapshot := nvs.Copy()
			nv := snapshot.PathMap["a.md"]
			assert.Equal(t, template.HTML("a.md@foo.com"), nv.HTMLForHost("foo.com"))
			assert.Equal(t, template.HTML("main"), nv.HTMLForHost("bar.com"))
			assert.Equal(t, template.HTML("main"), nv.HTMLForHost("other.com"))
			assert.Equal(t, template.HTML("main-domain"), nv.HTMLForHost(""))
		}()
	}
	wg.Wait()

	require.Equal(t, int64(2), renderer.renders.Load(), "one render per (note, host), shared by every copy")
}
