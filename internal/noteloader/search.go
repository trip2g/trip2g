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
const searchIndexSchemaVersion = "v2"

// langTextField builds an indexed, unstored text field mapping named `name`,
// analyzed with `analyzer`. Snippets are cut from Loader.searchTexts, so the
// index keeps only the term vectors they need. We index Title/Body under both a Russian-analyzed
// and an English-analyzed field so each language's BM25 lane stems correctly;
// the query in Search() is a per-field disjunction that matches via whichever
// language fits. This is independent of note.Lang, so it works even when the
// frontmatter language is missing or wrong.
func langTextField(name, analyzer string) *mapping.FieldMapping {
	fm := bleve.NewTextFieldMapping()
	fm.Name = name
	fm.Analyzer = analyzer
	fm.Store = false
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
	// The permalink is the document ID: a re-routed note must be re-indexed
	// under its new ID even when its text is unchanged.
	h.Write([]byte(note.Permalink))
	h.Write([]byte{0})
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

	// Every delete runs before any index write, so a permalink freed by one
	// note and taken by another in the same build keeps the new document.
	deleted, err := l.removeVanishedNotes(index, notes)
	if err != nil {
		return nil, err
	}

	changed, skipped, err := l.removeChangedNotes(index, notes)
	if err != nil {
		return nil, err
	}

	fresh := make(map[string]searchText, len(changed))
	for _, p := range changed {
		note := p.note
		content := noteContent{
			Title: note.Title,
			Body:  extractText(note.Ast(), note.Content),
			Hash:  hex.EncodeToString(p.hash[:]),
		}

		indexErr := index.Index(note.Permalink, content)
		if indexErr != nil {
			return nil, fmt.Errorf("failed to index note %s: %w", note.Permalink, indexErr)
		}

		l.contentHashes[note.PathID] = p.hash
		l.indexedPermalinks[note.PathID] = note.Permalink
		fresh[note.Permalink] = searchText{Title: []byte(content.Title), Body: []byte(content.Body)}
	}
	indexed := len(changed)

	texts := l.collectSearchTexts(notes, fresh)
	l.Lock()
	l.searchTexts = texts
	l.Unlock()

	l.log.Info("notes indexed",
		"indexed", indexed,
		"skipped", skipped,
		"deleted", deleted,
		"total", len(notes.List),
		"took", time.Since(startedAt).Seconds(),
	)

	return index, nil
}

// collectSearchTexts returns the text of every indexed note: fresh for the
// notes just indexed, the previous build's for unchanged ones, and extracted
// again for notes adopted from a persisted index, which this process never
// indexed. A new map, so Search can keep reading the old one meanwhile.
func (l *Loader) collectSearchTexts(notes *model.NoteViews, fresh map[string]searchText) map[string]searchText {
	l.Lock()
	previous := l.searchTexts
	l.Unlock()

	texts := make(map[string]searchText, len(notes.List))
	for _, note := range notes.List {
		if l.indexedPermalinks[note.PathID] != note.Permalink {
			continue
		}
		if t, ok := fresh[note.Permalink]; ok {
			texts[note.Permalink] = t
			continue
		}
		if t, ok := previous[note.Permalink]; ok {
			texts[note.Permalink] = t
			continue
		}
		texts[note.Permalink] = searchText{
			Title: []byte(note.Title),
			Body:  []byte(extractText(note.Ast(), note.Content)),
		}
	}
	return texts
}

// removeVanishedNotes deletes the documents of notes no longer in notes.
func (l *Loader) removeVanishedNotes(index bleve.Index, notes *model.NoteViews) (int, error) {
	current := make(map[int64]struct{}, len(notes.List))
	for _, note := range notes.List {
		current[note.PathID] = struct{}{}
	}

	deleted := 0
	for pathID := range l.contentHashes {
		if _, exists := current[pathID]; exists {
			continue
		}
		if err := l.removeFromIndex(index, pathID); err != nil {
			return 0, err
		}
		delete(l.contentHashes, pathID)
		deleted++
	}
	return deleted, nil
}

type pendingNote struct {
	note *model.NoteView
	hash [32]byte
}

// removeChangedNotes returns the notes whose document must be (re)written and
// deletes the documents that must go first: those of notes that became hidden
// or raw, and those under a permalink the note no longer has.
func (l *Loader) removeChangedNotes(index bleve.Index, notes *model.NoteViews) ([]pendingNote, int, error) {
	var changed []pendingNote
	skipped := 0
	for _, note := range notes.List {
		hash := contentHash(note)
		oldHash, exists := l.contentHashes[note.PathID]

		if exists && oldHash == hash {
			skipped++
			continue
		}

		// Raw files (.canvas, .base, .excalidraw) have no AST — nothing to index.
		hidden := note.ExcludeSearch || note.Ast() == nil
		if hidden || l.indexedPermalinks[note.PathID] != note.Permalink {
			if err := l.removeFromIndex(index, note.PathID); err != nil {
				return nil, 0, err
			}
		}
		if hidden {
			l.contentHashes[note.PathID] = hash
			continue
		}

		changed = append(changed, pendingNote{note: note, hash: hash})
	}
	return changed, skipped, nil
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

// Search returns up to size hits of queryString, skipping the first from.
func (l *Loader) Search(queryString string, from, size int) ([]model.SearchResult, error) {
	l.Lock()
	index, nvs, texts := l.searchIndex, l.nvs, l.searchTexts
	l.Unlock()

	if index == nil {
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

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.IncludeLocations = true
	searchRequest.From = from
	searchRequest.Size = size

	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := []model.SearchResult{}

	for _, hit := range searchResult.Hits {
		note, ok := nvs.Map[hit.ID]
		if !ok {
			continue
		}

		text := texts[hit.ID]
		titleSegments, titleSpans := highlightText(text.Title, hit.Locations, "Title", "Title_en")
		contentSegments, bodySpans := highlightText(text.Body, hit.Locations, "Body", "Body_en")

		results = append(results, model.SearchResult{
			NoteView:        note,
			URL:             note.Permalink,
			Score:           hit.Score,
			TitleSegments:   titleSegments,
			ContentSegments: contentSegments,
			MatchedTerms:    matchedTerms(text.Title, titleSpans, text.Body, bodySpans),
		})
	}

	return results, nil
}

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
