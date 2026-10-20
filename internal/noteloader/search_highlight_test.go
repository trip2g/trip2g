package noteloader

import (
	"strings"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/blevesearch/bleve/v2"
	htmlFilter "github.com/blevesearch/bleve/v2/analysis/char/html"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/stretchr/testify/require"
)

func searchOne(t *testing.T, query string, notes ...*model.NoteView) model.SearchResult {
	t.Helper()
	loader := &Loader{log: &logger.TestLogger{}}
	nvs := notesOf(notes...)
	index, err := loader.buildSearchIndex(nvs)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = nvs

	results, err := loader.Search(query, 0, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	return results[0]
}

func matches(segments []model.HighlightSegment) []string {
	var out []string
	for _, s := range segments {
		if s.Match {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestSearchHighlight_CyrillicStemmed(t *testing.T) {
	note := createNoteWithAST(1, "/capy", "Капибара дома", []byte("Капибары любят воду, а капибару любят все."))

	r := searchOne(t, "капибара", note)

	require.Equal(t, []model.HighlightSegment{
		{Text: "Капибара", Match: true},
		{Text: " дома"},
	}, r.TitleSegments)
	require.Equal(t, []model.HighlightSegment{
		{Text: "Капибары", Match: true},
		{Text: " любят воду, а "},
		{Text: "капибару", Match: true},
		{Text: " любят все."},
	}, r.ContentSegments)
	require.Equal(t, []string{"Капибара", "Капибары", "капибару"}, r.MatchedTerms)
}

func TestSearchHighlight_EnglishStemmed(t *testing.T) {
	note := createNoteWithAST(1, "/run", "Guide", []byte("Tips for running races."))

	r := searchOne(t, "run race", note)

	require.Equal(t, []string{"running", "races"}, matches(r.ContentSegments))
	require.Equal(t, []model.HighlightSegment{{Text: "Guide"}}, r.TitleSegments, "an unmatched title is shown whole")
}

func TestSearchHighlight_WindowWithMostDistinctTerms(t *testing.T) {
	filler := strings.Repeat("слово ", 60)
	body := "кот " + filler + "кот и собака " + filler
	note := createNoteWithAST(1, "/pets", "Pets", []byte(body))

	r := searchOne(t, "кот собака", note)

	require.Equal(t, []string{"кот", "собака"}, matches(r.ContentSegments))
	first := r.ContentSegments[0]
	last := r.ContentSegments[len(r.ContentSegments)-1]
	require.True(t, strings.HasPrefix(first.Text, highlightSeparator), "a fragment cut from the middle starts with the separator")
	require.True(t, strings.HasSuffix(last.Text, highlightSeparator), "and ends with it")
	require.LessOrEqual(t, len([]rune(model.SegmentsText(r.ContentSegments))), highlightFragmentRunes+2)
}

func TestSearchHighlight_NoBodyMatchShowsTheStart(t *testing.T) {
	body := strings.Repeat("текст ", 50)
	note := createNoteWithAST(1, "/t", "Уникальный заголовок", []byte(body))

	r := searchOne(t, "уникальный", note)

	require.Equal(t, []string{"Уникальный"}, matches(r.TitleSegments))
	require.Len(t, r.ContentSegments, 1)
	require.False(t, r.ContentSegments[0].Match)
	require.True(t, strings.HasPrefix(r.ContentSegments[0].Text, "текст"), "no separator at the start of the text")
	require.True(t, strings.HasSuffix(r.ContentSegments[0].Text, highlightSeparator))
}

func TestSearchHighlight_EscapedInHTML(t *testing.T) {
	note := createNoteWithAST(1, "/esc", "Esc", []byte("Use `a < b && c` with капибара."))

	r := searchOne(t, "капибара", note)

	require.Equal(t, "Use a &lt; b &amp;&amp; c with <mark>капибара</mark>.", model.HighlightHTML(r.ContentSegments))
}

// The deprecated html fields must read as they did when bleve's own
// highlighter built them from stored fields.
func TestSearchHighlight_MatchesBleveHighlighter(t *testing.T) {
	long := strings.Repeat("Обычное предложение без нужных слов. ", 12)
	notes := []*model.NoteView{
		createNoteWithAST(1, "/a", "Сад и огород", []byte("В саду растут яблони. "+long+"Огород рядом с садом, а в огороде <растёт> & картофель.")),
		createNoteWithAST(2, "/b", "Заметки о саде", []byte(long+"Сады цветут весной.")),
		createNoteWithAST(3, "/c", "Garden", []byte("The garden is green. "+long)),
	}
	queries := []string{"сад", "огород", "сад огород", "garden", "весной"}

	loader := &Loader{log: &logger.TestLogger{}}
	nvs := notesOf(notes...)
	index, err := loader.buildSearchIndex(nvs)
	require.NoError(t, err)
	loader.searchIndex = index
	loader.nvs = nvs

	stored := bleveStoredIndex(t, notes)

	for _, q := range queries {
		results, searchErr := loader.Search(q, 0, 10)
		require.NoError(t, searchErr)
		require.NotEmpty(t, results, q)

		mq := bleve.NewMatchQuery(q)
		mq.SetOperator(bleveQuery.MatchQueryOperatorAnd)
		req := bleve.NewSearchRequest(mq)
		req.Highlight = bleve.NewHighlightWithStyle(htmlFilter.Name)
		req.Highlight.AddField("Title")
		req.Highlight.AddField("Body")
		want, searchErr := stored.Search(req)
		require.NoError(t, searchErr)
		require.Len(t, want.Hits, len(results), q)

		byURL := map[string]model.SearchResult{}
		for _, r := range results {
			byURL[r.URL] = r
		}
		for _, hit := range want.Hits {
			got, ok := byURL[hit.ID]
			require.True(t, ok, "%s: %s", q, hit.ID)
			require.Equal(t, hit.Fragments["Title"][0], model.HighlightHTML(got.TitleSegments), "%s: %s title", q, hit.ID)
			require.Equal(t, hit.Fragments["Body"][0], model.HighlightHTML(got.ContentSegments), "%s: %s body", q, hit.ID)
		}
	}
}

// bleveStoredIndex is the index as it was before snippets moved out of it:
// stored ru-analyzed fields that bleve highlights itself.
func bleveStoredIndex(t *testing.T, notes []*model.NoteView) bleve.Index {
	t.Helper()
	doc := bleve.NewDocumentMapping()
	doc.Dynamic = false
	doc.AddFieldMappingsAt("Title", langTextField("Title", "ru"))
	doc.AddFieldMappingsAt("Body", langTextField("Body", "ru"))
	doc.Properties["Title"].Fields[0].Store = true
	doc.Properties["Body"].Fields[0].Store = true

	m := bleve.NewIndexMapping()
	m.DefaultMapping = doc
	m.DefaultAnalyzer = "ru"
	index, err := bleve.NewMemOnly(m)
	require.NoError(t, err)

	for _, n := range notes {
		content := noteContent{Title: n.Title, Body: extractText(n.Ast(), n.Content)}
		require.NoError(t, index.Index(n.Permalink, content))
	}
	return index
}
