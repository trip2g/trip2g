package sitesearch

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/yuin/goldmark/ast"

	"github.com/blevesearch/bleve/v2"
	htmlFilter "github.com/blevesearch/bleve/v2/analysis/char/html"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/ru"

	"trip2g/internal/graph/model"
	"trip2g/internal/logger"
	appmodel "trip2g/internal/model"
)

type Env interface {
	LiveNoteViews() *appmodel.NoteViews
	Logger() logger.Logger
}

type noteContent struct {
	Title string
	Body  string
}

func Resolve(ctx context.Context, env Env, input model.SearchInput) (*model.SearchConnection, error) {
	log := logger.WithPrefix(env.Logger(), "sitesearch:")

	nvs := env.LiveNoteViews()
	notes := nvs.VisibleList()

	documentMapping := bleve.NewDocumentMapping()

	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Analyzer = "ru"
	titleFieldMapping.Store = true
	titleFieldMapping.Index = true
	documentMapping.AddFieldMappingsAt("Title", titleFieldMapping)

	bodyFieldMapping := bleve.NewTextFieldMapping()
	bodyFieldMapping.Analyzer = "ru"
	bodyFieldMapping.Store = true
	bodyFieldMapping.Index = true
	documentMapping.AddFieldMappingsAt("Body", bodyFieldMapping)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("note", documentMapping)
	mapping.DefaultAnalyzer = "ru"

	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		panic(err)
	}

	startedAt := time.Now()

	for _, note := range notes {
		content := noteContent{
			Title: note.Title,
			Body:  extractText(note.Ast(), note.Content),
		}

		indexErr := index.Index(note.Permalink, content)
		if indexErr != nil {
			panic(indexErr)
		}
	}

	log.Info("notes indexed", "count", len(notes), "took", time.Since(startedAt).Seconds())

	queryString := strings.TrimSpace(input.Query)
	conn := model.SearchConnection{}

	if len(queryString) < 3 {
		return &conn, nil
	}

	query := bleve.NewMatchQuery(queryString)

	highlight := bleve.NewHighlightWithStyle(htmlFilter.Name)
	highlight.AddField("Title")
	highlight.AddField("Body")

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.IncludeLocations = true
	searchRequest.Highlight = highlight
	searchRequest.Fields = []string{"*"}
	searchRequest.Size = 20

	searchResult, err := index.Search(searchRequest)
	if err != nil {
		panic(err)
	}

	for _, hit := range searchResult.Hits {
		note, ok := nvs.Map[hit.ID]
		if !ok {
			continue
		}

		publicNote := model.ConvertNoteToPublic(note)

		res := model.SearchResult{
			Document: publicNote,
			URL:      note.Permalink,
		}

		for field, fragments := range hit.Fragments {
			if field == "Title" && len(fragments) > 0 {
				res.HighlightedTitle = &fragments[0]
				continue
			}

			if field == "Body" {
				res.HighlightedContent = fragments
			}
		}

		conn.Nodes = append(conn.Nodes, res)
	}

	return &conn, nil
}

// extractText extracts plain text from a Markdown AST.
// This version is optimized for getting the minimal text content
// without complex formatting like newlines and indentation.
func extractText(doc ast.Node, src []byte) string {
	var buf bytes.Buffer
	var lastNode ast.Node

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			// Add a space after certain block-level nodes to prevent words from merging.
			switch n.Kind() {
			case ast.KindHeading, ast.KindParagraph, ast.KindBlockquote, ast.KindListItem:
				if lastNode != nil {
					if lastNode.Kind() == ast.KindText || lastNode.Kind() == ast.KindCodeSpan {
						buf.WriteString(" ")
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
