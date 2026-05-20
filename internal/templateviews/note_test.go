package templateviews

import (
	"html/template"
	"testing"

	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestNote_HTMLString_MainDomainDomainHTML(t *testing.T) {
	// Regression test: HTMLString() must use DomainHTML[""] on the main domain
	// (domainHost == ""). Previously, the guard `domainHost != ""` caused it
	// to always return nv.HTML for main-domain requests, ignoring DomainHTML[""].
	nv := &model.NoteView{
		HTML: template.HTML(`<a href="/extra">extra</a>`),
		DomainHTML: map[string]template.HTML{
			"": template.HTML(`<a href="https://extra.trip2g.com/">extra</a>`),
		},
	}

	// Main domain context (domainHost == "").
	note := NewNoteWithDomain(nv, "")
	require.Equal(t, `<a href="https://extra.trip2g.com/">extra</a>`, note.HTMLString(),
		"main domain: HTMLString should use DomainHTML[\"\"] not nv.HTML")

	// Custom domain context — should use custom domain HTML when available.
	nv2 := &model.NoteView{
		HTML: template.HTML(`<a href="/extra">extra</a>`),
		DomainHTML: map[string]template.HTML{
			"foo.com": template.HTML(`<a href="/custom-path">extra</a>`),
		},
	}
	note2 := NewNoteWithDomain(nv2, "foo.com")
	require.Equal(t, `<a href="/custom-path">extra</a>`, note2.HTMLString(),
		"custom domain: HTMLString should use DomainHTML[domainHost]")

	// No DomainHTML — falls back to nv.HTML.
	nv3 := &model.NoteView{
		HTML: template.HTML(`<a href="/plain">plain</a>`),
	}
	note3 := NewNoteWithDomain(nv3, "")
	require.Equal(t, `<a href="/plain">plain</a>`, note3.HTMLString(),
		"no DomainHTML: HTMLString should return nv.HTML")
}

func TestNote_PermalinkEncoded(t *testing.T) {
	nv := &model.NoteView{Permalink: "/поиск_астрономия"}
	note := NewNote(nv)
	require.Equal(t, "/%D0%BF%D0%BE%D0%B8%D1%81%D0%BA_%D0%B0%D1%81%D1%82%D1%80%D0%BE%D0%BD%D0%BE%D0%BC%D0%B8%D1%8F", note.PermalinkEncoded())

	nv2 := &model.NoteView{Permalink: "/hello_world"}
	note2 := NewNote(nv2)
	require.Equal(t, "/hello_world", note2.PermalinkEncoded())
}

func TestNote_FormSpecJSON(t *testing.T) {
	t.Run("no form frontmatter returns empty", func(t *testing.T) {
		nv := &model.NoteView{VersionID: 42, RawMeta: map[string]interface{}{"title": "x"}}
		require.Empty(t, NewNote(nv).FormSpecJSON())
	})

	t.Run("single form maps to empty key", func(t *testing.T) {
		nv := &model.NoteView{
			VersionID: 42,
			RawMeta: map[string]interface{}{
				"form": map[string]interface{}{
					"can_submit": "guest",
					"fields": []interface{}{
						map[string]interface{}{"name": "email", "type": "email", "required": true},
					},
				},
			},
		}
		got := NewNote(nv).FormSpecJSON()
		require.Contains(t, got, `"note_version_id":42`)
		require.Contains(t, got, `"forms":{"":`)
		require.Contains(t, got, `"can_submit":"guest"`)
		require.Contains(t, got, `"name":"email"`)
	})

	t.Run("forms map keeps named keys", func(t *testing.T) {
		nv := &model.NoteView{
			VersionID: 7,
			RawMeta: map[string]interface{}{
				"forms": map[string]interface{}{
					"contact": map[string]interface{}{
						"can_submit": "guest",
						"fields": []interface{}{
							map[string]interface{}{"name": "email", "type": "email"},
						},
					},
				},
			},
		}
		got := NewNote(nv).FormSpecJSON()
		require.Contains(t, got, `"note_version_id":7`)
		require.Contains(t, got, `"contact":`)
	})
}
