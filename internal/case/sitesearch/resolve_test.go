package sitesearch

import (
	"testing"

	"github.com/stretchr/testify/require"

	appmodel "trip2g/internal/model"
)

func makeResult(url string, score float64, title string) appmodel.SearchResult {
	return appmodel.SearchResult{
		URL:   url,
		Score: score,
		NoteView: &appmodel.NoteView{
			Permalink: url,
			Title:     title,
		},
	}
}

func TestMergeResults_EmptyVectorReturnsText(t *testing.T) {
	text := []appmodel.SearchResult{
		makeResult("/a", 1.0, "A"),
		makeResult("/b", 0.5, "B"),
	}

	result := mergeResults(text, nil)

	require.Equal(t, text, result)
}

func TestMergeResults_RRFScoreFormula(t *testing.T) {
	// Single doc at rank 0 (1-indexed: 1) in text list only.
	// RRF score = 1 / (60 + 1) = 1/61 ≈ 0.01639
	text := []appmodel.SearchResult{makeResult("/a", 9.9, "A")}
	vector := []appmodel.SearchResult{makeResult("/b", 0.99, "B")}

	result := mergeResults(text, vector)

	require.Len(t, result, 2)

	scoreA := result[0].Score
	scoreB := result[1].Score

	expected := 1.0 / float64(rrfK+1)
	require.InDelta(t, expected, scoreA, 1e-10)
	require.InDelta(t, expected, scoreB, 1e-10)
}

func TestMergeResults_DocInBothListsGetsCombinedScore(t *testing.T) {
	// /shared is rank 0 in text, rank 0 in vector → score = 2*(1/61)
	// /text-only is rank 1 in text → score = 1/(61+1) = 1/62
	// /vec-only is rank 1 in vector → score = 1/62
	text := []appmodel.SearchResult{
		makeResult("/shared", 5.0, "Shared"),
		makeResult("/text-only", 3.0, "Text Only"),
	}
	vector := []appmodel.SearchResult{
		makeResult("/shared", 0.9, "Shared"),
		makeResult("/vec-only", 0.7, "Vec Only"),
	}

	result := mergeResults(text, vector)

	require.Len(t, result, 3)

	// /shared should be first (highest combined score)
	require.Equal(t, "/shared", result[0].URL)

	expectedShared := 2.0 / float64(rrfK+1)
	require.InDelta(t, expectedShared, result[0].Score, 1e-10)

	// /text-only and /vec-only both have 1/62, in any order
	require.InDelta(t, 1.0/float64(rrfK+2), result[1].Score, 1e-10)
	require.InDelta(t, 1.0/float64(rrfK+2), result[2].Score, 1e-10)
}

func TestMergeResults_TextHighlightsPreservedForSharedDoc(t *testing.T) {
	highlightedTitle := "Найдено: <mark>Go</mark>"
	highlighted := []string{"контент с <mark>Go</mark> выделением"}

	text := []appmodel.SearchResult{{
		URL:                "/shared",
		Score:              1.0,
		HighlightedTitle:   &highlightedTitle,
		HighlightedContent: highlighted,
		NoteView:           &appmodel.NoteView{Permalink: "/shared", Title: "Go"},
	}}
	vector := []appmodel.SearchResult{makeResult("/shared", 0.9, "Go")}

	result := mergeResults(text, vector)

	require.Len(t, result, 1)
	require.Equal(t, "/shared", result[0].URL)
	// Text highlights preserved (not overwritten by vector result)
	require.NotNil(t, result[0].HighlightedTitle)
	require.Equal(t, highlightedTitle, *result[0].HighlightedTitle)
	require.Equal(t, highlighted, result[0].HighlightedContent)
}

func TestMergeResults_VectorOnlyResultGetsSnippet(t *testing.T) {
	text := []appmodel.SearchResult{makeResult("/text", 1.0, "Text")}
	vector := []appmodel.SearchResult{{
		URL:   "/vec-only",
		Score: 0.95,
		NoteView: &appmodel.NoteView{
			Permalink: "/vec-only",
			Title:     "Vector Note",
			Content:   []byte("This is the content of the vector-only note."),
		},
	}}

	result := mergeResults(text, vector)

	require.Len(t, result, 2)

	var vecResult *appmodel.SearchResult
	for i := range result {
		if result[i].URL == "/vec-only" {
			vecResult = &result[i]
			break
		}
	}

	require.NotNil(t, vecResult)
	require.NotNil(t, vecResult.HighlightedTitle)
	require.Equal(t, "Vector Note", *vecResult.HighlightedTitle)
	require.NotEmpty(t, vecResult.HighlightedContent)
}

func TestMergeResults_LimitedTo20(t *testing.T) {
	// 15 text + 15 vector, 5 shared = 25 unique docs
	var text, vector []appmodel.SearchResult
	for i := range 15 {
		url := "/text-%d"
		_ = i
		text = append(text, makeResult(url, float64(15-i), "T"))
	}
	for i := range 15 {
		url := "/vec-%d"
		_ = i
		vector = append(vector, makeResult(url, float64(15-i)*0.1, "V"))
	}

	result := mergeResults(text, vector)

	require.LessOrEqual(t, len(result), 20)
}

func TestMergeResults_RankOrderMatters(t *testing.T) {
	// /a is rank 0 in text, /b is rank 1 in text
	// With no vector results, /a should score higher
	text := []appmodel.SearchResult{
		makeResult("/a", 10.0, "A"),
		makeResult("/b", 9.0, "B"),
	}
	vector := []appmodel.SearchResult{makeResult("/x", 0.5, "X")} // unrelated

	result := mergeResults(text, vector)

	// Find /a and /b positions
	var posA, posB int
	for i, r := range result {
		if r.URL == "/a" {
			posA = i
		}
		if r.URL == "/b" {
			posB = i
		}
	}
	require.Less(t, posA, posB, "/a (rank 0 in text) should rank above /b (rank 1 in text)")
}
