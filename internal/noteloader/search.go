package noteloader

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	// Hash is the note's contentHash in hex. It is stored, never indexed, and
	// exists so a persisted index can say what it already holds: after a
	// restart the in-memory contentHashes map is empty, and without this the
	// loader could neither skip unchanged notes nor notice notes deleted while
	// the process was down. See adoptPersistedIndex.
	Hash string
}

// searchIndexSchemaVersion changes whenever the mapping below changes in a way
// that makes an existing on-disk index wrong (new field, different analyzer).
// The index lives in a directory named after it, and createSearchIndex deletes
// directories from other versions, so a bump rebuilds instead of silently
// serving results from a stale schema.
const searchIndexSchemaVersion = "v1"

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

	// Stored but not indexed: it must never match a query, only travel with the
	// document so a reopened index can be compared against the database.
	hashField := bleve.NewKeywordFieldMapping()
	hashField.Name = "Hash"
	hashField.Store = true
	hashField.Index = false
	hashField.IncludeInAll = false
	hashField.IncludeTermVectors = false
	documentMapping.AddFieldMappingsAt("Hash", hashField)

	// Use this as the DEFAULT mapping: notes are indexed as plain structs with no
	// type field, so a mapping registered under a named type would never apply
	// (bleve would fall back to the dynamic default analyzer for everything).
	mapping := bleve.NewIndexMapping()
	mapping.DefaultMapping = documentMapping
	mapping.DefaultAnalyzer = "ru"

	if l.searchIndexPath == "" {
		return bleve.NewMemOnly(mapping)
	}

	// One directory per loader version ("live", "latest", ...) so the two
	// loaders never share an index, and one below it per schema version.
	dir := filepath.Join(l.searchIndexPath, l.version, searchIndexSchemaVersion)
	if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
		return nil, fmt.Errorf("failed to create search index directory: %w", err)
	}
	l.removeStaleIndexVersions(filepath.Dir(dir))

	index, err := bleve.Open(dir)
	switch {
	case err == nil:
		l.adoptOnNextBuild = true
		l.log.Info("search index opened from disk", "path", dir)
		return index, nil
	case errors.Is(err, bleve.ErrorIndexPathDoesNotExist):
		// First run for this schema version: build it below.
	default:
		// Deliberately NOT deleting the directory here. The obvious reading of
		// "cannot open" is "corrupt, rebuild it", and it is wrong: a
		// zero-downtime handoff runs the old and the new instance at the same
		// time, the old one holds the index lock, and rebuilding would destroy a
		// live index out from under it. Fall back to memory for this process
		// instead — correct, slower, and loud. A genuinely corrupt index is a
		// human's call: delete the directory and restart.
		l.log.Error("search index could not be opened, falling back to the in-memory index",
			"path", dir, "error", err)
		return bleve.NewMemOnly(mapping)
	}

	index, err = bleve.New(dir, mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to create on-disk search index at %s: %w", dir, err)
	}
	l.log.Info("search index created on disk", "path", dir)
	return index, nil
}

// removeStaleIndexVersions deletes index directories written by another schema
// version. Left alone they would sit on disk forever, one dead copy per bump.
func (l *Loader) removeStaleIndexVersions(parent string) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == searchIndexSchemaVersion {
			continue
		}
		stale := filepath.Join(parent, e.Name())
		if rmErr := os.RemoveAll(stale); rmErr != nil {
			l.log.Warn("failed to remove stale search index", "path", stale, "error", rmErr)
			continue
		}
		l.log.Info("removed search index of an older schema version", "path", stale)
	}
}

func contentHash(note *model.NoteView) [32]byte {
	h := sha256.New()
	h.Write([]byte(note.Title))
	h.Write(note.Content)
	// Visibility is part of the hash: toggling search:false alone must not be
	// skipped as "unchanged".
	if note.ExcludeSearch {
		h.Write([]byte{1})
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// removeFromIndex deletes a previously indexed document, if any.
func (l *Loader) removeFromIndex(index bleve.Index, pathID int64) error {
	permalink, ok := l.indexedPermalinks[pathID]
	if !ok {
		return nil
	}
	if err := index.Delete(permalink); err != nil {
		return fmt.Errorf("failed to delete note %s from index: %w", permalink, err)
	}
	delete(l.indexedPermalinks, pathID)
	return nil
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
	if l.indexedPermalinks == nil {
		l.indexedPermalinks = make(map[int64]string)
	}

	if l.adoptOnNextBuild {
		if err := l.adoptPersistedIndex(index, notes); err != nil {
			return nil, err
		}
		l.adoptOnNextBuild = false
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

		// Raw files (.canvas, .base, .excalidraw) have no AST — nothing to index.
		// A note that became hidden or raw may still be in the index — delete it.
		if note.ExcludeSearch || note.Ast() == nil {
			if delErr := l.removeFromIndex(index, note.PathID); delErr != nil {
				return nil, delErr
			}
			l.contentHashes[note.PathID] = hash
			continue
		}

		content := noteContent{
			Title: note.Title,
			Body:  extractText(note.Ast(), note.Content),
			Hash:  hex.EncodeToString(hash[:]),
		}

		indexErr := index.Index(note.Permalink, content)
		if indexErr != nil {
			return nil, fmt.Errorf("failed to index note %s: %w", note.Permalink, indexErr)
		}

		l.contentHashes[note.PathID] = hash
		l.indexedPermalinks[note.PathID] = note.Permalink
		indexed++
	}

	// Remove deleted notes from index and hashes
	deleted := 0
	for pathID := range l.contentHashes {
		if _, exists := currentPathIDs[pathID]; !exists {
			if delErr := l.removeFromIndex(index, pathID); delErr != nil {
				return nil, delErr
			}
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

// adoptPersistedIndex teaches a freshly reopened on-disk index to the loader:
// it reads back the hash stored with every document and rebuilds the two maps
// that drive incremental indexing, then deletes documents whose note no longer
// exists.
//
// Both halves matter. Without the hashes every note would be re-indexed on
// every boot, which is the work the on-disk index exists to avoid. Without the
// deletion pass a note removed while the process was down would stay in the
// index forever, because the deletion check below walks contentHashes, and on
// a fresh process that map starts empty.
func (l *Loader) adoptPersistedIndex(index bleve.Index, notes *model.NoteViews) error {
	count, err := index.DocCount()
	if err != nil {
		return fmt.Errorf("failed to count documents in the persisted index: %w", err)
	}
	if count == 0 {
		return nil
	}

	req := bleve.NewSearchRequest(bleve.NewMatchAllQuery())
	req.Size = int(count)
	req.Fields = []string{"Hash"}
	result, err := index.Search(req)
	if err != nil {
		return fmt.Errorf("failed to read back the persisted index: %w", err)
	}

	stored := make(map[string][32]byte, len(result.Hits))
	for _, hit := range result.Hits {
		raw, ok := hit.Fields["Hash"].(string)
		if !ok {
			continue // pre-hash document: treat as unknown, it will be re-indexed
		}
		decoded, decErr := hex.DecodeString(raw)
		if decErr != nil || len(decoded) != sha256.Size {
			continue
		}
		var h [32]byte
		copy(h[:], decoded)
		stored[hit.ID] = h
	}

	adopted := 0
	for _, note := range notes.List {
		h, ok := stored[note.Permalink]
		if !ok {
			continue
		}
		l.contentHashes[note.PathID] = h
		l.indexedPermalinks[note.PathID] = note.Permalink
		delete(stored, note.Permalink)
		adopted++
	}

	// Whatever is left belongs to no current note: the note was deleted, renamed
	// or hidden while this process was not running.
	orphans := 0
	for permalink := range stored {
		if delErr := index.Delete(permalink); delErr != nil {
			return fmt.Errorf("failed to delete orphaned document %s: %w", permalink, delErr)
		}
		orphans++
	}

	l.log.Info("persisted search index adopted", "documents", count, "adopted", adopted, "orphans_removed", orphans)
	return nil
}

func (l *Loader) Search(queryString string) ([]model.SearchResult, error) {
	if l.searchIndex == nil {
		return nil, ErrSearchNotAvailable
	}

	// Per-field disjunction: the query is analyzed with each field's own analyzer
	// (ru for Title/Body, en for Title_en/Body_en) and a doc matches via whichever
	// language fits. Within a field, terms are AND-ed (same precision as before).
	matchField := func(field string) *bleveQuery.MatchQuery {
		mq := bleve.NewMatchQuery(queryString)
		mq.SetField(field)
		mq.SetOperator(bleveQuery.MatchQueryOperatorAnd)
		return mq
	}
	query := bleve.NewDisjunctionQuery(
		matchField("Title"), matchField("Title_en"),
		matchField("Body"), matchField("Body_en"),
	)

	highlight := bleve.NewHighlightWithStyle(htmlFilter.Name)
	highlight.AddField("Title")
	highlight.AddField("Title_en")
	highlight.AddField("Body")
	highlight.AddField("Body_en")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.IncludeLocations = true
	searchRequest.Highlight = highlight
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 20

	searchResult, err := l.searchIndex.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := []model.SearchResult{}

	for _, hit := range searchResult.Hits {
		note, ok := l.nvs.Map[hit.ID]
		if !ok {
			continue
		}

		result := model.SearchResult{
			NoteView: note,
			URL:      note.Permalink,
			Score:    hit.Score,
		}

		// hit.Fragments is a map → Go randomizes iteration order, and the two
		// analyzer views of a field (e.g. Body vs Body_en) can yield different
		// highlighted fragments. Select fields in a FIXED priority order so
		// identical queries produce identical highlighting (deterministic snippets).
		// Title and Title_en are two analyzer views of the same source field;
		// likewise Body and Body_en.
		for _, field := range []string{"Title", "Title_en"} {
			if frags := hit.Fragments[field]; len(frags) > 0 {
				result.HighlightedTitle = &frags[0]
				break
			}
		}
		for _, field := range []string{"Body", "Body_en"} {
			if frags := hit.Fragments[field]; len(frags) > 0 {
				result.HighlightedContent = frags
				break
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
