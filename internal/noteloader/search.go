package noteloader

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
	"trip2g/internal/model"

	htmlFilter "github.com/blevesearch/bleve/v2/analysis/char/html"

	_ "github.com/blevesearch/bleve/v2/analysis/lang/en"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ru"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	bleveQuery "github.com/blevesearch/bleve/v2/search/query"
	"github.com/yuin/goldmark/ast"
)

var ErrSearchNotAvailable = errors.New("search index is not available")

type noteContent struct {
	Title string
	Body  string
}

// langTextField builds a stored, indexed text field mapping named `name`,
// analyzed with `analyzer`. We index Title/Body under both a Russian-analyzed
// and an English-analyzed field so each language's BM25 lane stems correctly;
// the query in Search() is a per-field disjunction that matches via whichever
// language fits. This is independent of note.Lang, so it works even when the
// frontmatter language is missing or wrong.
func langTextField(name, analyzer string) *mapping.FieldMapping {
	fm := bleve.NewTextFieldMapping()
	fm.Name = name
	fm.Analyzer = analyzer
	fm.Store = true
	fm.Index = true
	return fm
}

func (l *Loader) createSearchIndex() (bleve.Index, error) {
	documentMapping := bleve.NewDocumentMapping()

	documentMapping.Dynamic = false
	documentMapping.AddFieldMappingsAt("Title",
		langTextField("Title", "ru"),
		langTextField("Title_en", "en"),
	)
	documentMapping.AddFieldMappingsAt("Body",
		langTextField("Body", "ru"),
		langTextField("Body_en", "en"),
	)

	// Use this as the DEFAULT mapping: notes are indexed as plain structs with no
	// type field, so a mapping registered under a named type would never apply
	// (bleve would fall back to the dynamic default analyzer for everything).
	mapping := bleve.NewIndexMapping()
	mapping.DefaultMapping = documentMapping
	mapping.DefaultAnalyzer = "ru"

	return bleve.NewMemOnly(mapping)
}

func contentHash(note *model.NoteView) [32]byte {
	h := sha256.New()
	h.Write([]byte(note.Title))
	h.Write(note.Content)
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func (l *Loader) buildSearchIndex(notes *model.NoteViews) (bleve.Index, error) {
	startedAt := time.Now()

	// Reuse existing index or create new one
	index := l.searchIndex
	if index == nil {
		var err error
		index, err = l.createSearchIndex()
		if err != nil {
			return nil, fmt.Errorf("failed to create bleve index: %w", err)
		}
	}

	// Initialize content hashes map if needed
	if l.contentHashes == nil {
		l.contentHashes = make(map[int64][32]byte)
	}

	// Track current PathIDs to detect deleted notes
	currentPathIDs := make(map[int64]struct{})
	indexed := 0
	skipped := 0

	for _, note := range notes.List {
		currentPathIDs[note.PathID] = struct{}{}

		hash := contentHash(note)
		oldHash, exists := l.contentHashes[note.PathID]

		// Skip if content hasn't changed
		if exists && oldHash == hash {
			skipped++
			continue
		}

		if note.ExcludeSearch {
			continue
		}

		// Raw files (.canvas, .base, .excalidraw) have no AST — skip indexing.
		// They aren't markdown, no meaningful text to search anyway.
		if note.Ast() == nil {
			continue
		}

		content := noteContent{
			Title: note.Title,
			Body:  extractText(note.Ast(), note.Content),
		}

		indexErr := index.Index(note.Permalink, content)
		if indexErr != nil {
			return nil, fmt.Errorf("failed to index note %s: %w", note.Permalink, indexErr)
		}

		l.contentHashes[note.PathID] = hash
		indexed++
	}

	// Remove deleted notes from index and hashes
	deleted := 0
	for pathID := range l.contentHashes {
		if _, exists := currentPathIDs[pathID]; !exists {
			// Note was deleted, but we don't have permalink to delete from index
			// Just remove from hashes map
			delete(l.contentHashes, pathID)
			deleted++
		}
	}

	l.log.Info("notes indexed",
		"indexed", indexed,
		"skipped", skipped,
		"deleted", deleted,
		"total", len(notes.List),
		"took", time.Since(startedAt).Seconds(),
	)

	return index, nil
}

// andOrFallbackThreshold is the minimum number of AND hits below which we fall
// back to OR and merge results. AND gives higher precision for multi-term queries,
// but when the corpus is small or the query contains rare/inflected terms, AND can
// return zero or near-zero results. A threshold of 3 means: if AND finds fewer than
// 3 documents, re-run with OR and append OR-only hits after the AND hits.
const andOrFallbackThreshold = 3

func (l *Loader) Search(queryString string) ([]model.SearchResult, error) {
	if l.searchIndex == nil {
		return nil, ErrSearchNotAvailable
	}

	// buildQuery constructs a per-field disjunction with the given term operator.
	// The query is analyzed with each field's own analyzer (ru / en) and a doc
	// matches via whichever language fits.
	buildQuery := func(op bleveQuery.MatchQueryOperator) *bleveQuery.DisjunctionQuery {
		matchField := func(field string) *bleveQuery.MatchQuery {
			mq := bleve.NewMatchQuery(queryString)
			mq.SetField(field)
			mq.SetOperator(op)
			return mq
		}
		return bleve.NewDisjunctionQuery(
			matchField("Title"), matchField("Title_en"),
			matchField("Body"), matchField("Body_en"),
		)
	}

	highlight := bleve.NewHighlightWithStyle(htmlFilter.Name)
	highlight.AddField("Title")
	highlight.AddField("Title_en")
	highlight.AddField("Body")
	highlight.AddField("Body_en")

	runSearch := func(query bleveQuery.Query) (*bleve.SearchResult, error) {
		req := bleve.NewSearchRequest(query)
		req.IncludeLocations = true
		req.Highlight = highlight
		req.Fields = []string{"*"}
		req.Size = 20
		return l.searchIndex.Search(req)
	}

	// First pass: AND — highest precision, all query terms must appear in the field.
	andResult, err := runSearch(buildQuery(bleveQuery.MatchQueryOperatorAnd))
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// AND→OR fallback: if AND returns fewer than andOrFallbackThreshold hits, re-run
	// with OR to recover recall. OR hits are appended after AND hits (AND first /
	// higher precision), deduplicated by document ID.
	hits := andResult.Hits
	if len(hits) < andOrFallbackThreshold {
		orResult, orErr := runSearch(buildQuery(bleveQuery.MatchQueryOperatorOr))
		if orErr == nil {
			andIDs := make(map[string]struct{}, len(hits))
			for _, h := range hits {
				andIDs[h.ID] = struct{}{}
			}
			for _, h := range orResult.Hits {
				if _, seen := andIDs[h.ID]; !seen {
					hits = append(hits, h)
				}
			}
		}
	}

	results := []model.SearchResult{}

	for _, hit := range hits {
		note, ok := l.nvs.Map[hit.ID]
		if !ok {
			continue
		}

		result := model.SearchResult{
			NoteView: note,
			URL:      note.Permalink,
			Score:    hit.Score,
		}

		for field, fragments := range hit.Fragments {
			if len(fragments) == 0 {
				continue
			}

			// Title and Title_en are two analyzer views of the same source field;
			// likewise Body and Body_en. Map both to the same result field.
			if field == "Title" || field == "Title_en" {
				if result.HighlightedTitle == nil {
					result.HighlightedTitle = &fragments[0]
				}
				continue
			}

			if field == "Body" || field == "Body_en" {
				if len(result.HighlightedContent) == 0 {
					result.HighlightedContent = fragments
				}
				continue
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// I don't know how to change the highlight tags in bleve here
// search/highlight/format/html/html.go
// func replaceMarkToEmphasis(s string) string {
// 	s = strings.ReplaceAll(s, "<mark>", "<em>")
// 	s = strings.ReplaceAll(s, "</mark>", "</em>")
// 	return s
// }

// extractText extracts plain text from a Markdown AST.
// This version is optimized for getting the minimal text content
// without complex formatting like newlines and indentation.
//
//nolint:gocognit // ast traversal is always complex
func extractText(doc ast.Node, src []byte) string {
	var buf bytes.Buffer
	var lastNode ast.Node

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			// Add a newline after block-level nodes so snippets stay readable.
			switch n.Kind() {
			case ast.KindHeading, ast.KindParagraph, ast.KindBlockquote, ast.KindListItem:
				if lastNode != nil {
					if lastNode.Kind() == ast.KindText || lastNode.Kind() == ast.KindCodeSpan {
						buf.WriteString("\n")
					}
				}
			}
			lastNode = n
			return ast.WalkContinue, nil
		}

		// Handle nodes on entry.
		switch node := n.(type) {
		case *ast.Text:
			buf.Write(node.Segment.Value(src))
		case *ast.CodeSpan:
			// The text for CodeSpan is in its children.
			// Walk its children to get the content.
			_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering {
					if textNode, ok := n.(*ast.Text); ok {
						buf.Write(textNode.Segment.Value(src))
					}
				}
				return ast.WalkContinue, nil
			})
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			// Extract text from code blocks line by line.
			lines := node.Lines()
			for i := range lines.Len() {
				line := lines.At(i)
				buf.Write(line.Value(src))
			}
		case *ast.Image:
			// For images, extract the alt text from its children.
			// The alt text is contained in *ast.Text nodes.
			_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering {
					if textNode, ok := n.(*ast.Text); ok {
						buf.Write(textNode.Segment.Value(src))
					}
				}
				return ast.WalkContinue, nil
			})
			return ast.WalkSkipChildren, nil
		case *ast.ThematicBreak, *ast.List, *ast.Link, *ast.Document:
			// These are container nodes; their children will be handled automatically.
			return ast.WalkContinue, nil
		}

		lastNode = n
		return ast.WalkContinue, nil
	})

	return strings.TrimSpace(buf.String())
}
