package model

import (
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/yuin/goldmark/ast"
)

type NoteView struct {
	Path  string
	Title string

	PathID int64

	Content []byte
	HTML    template.HTML
	ast     ast.Node // hide from JSON

	Permalink string
	Free      bool // without the paywall

	InLinks map[string]struct{}
	RawMeta map[string]interface{}

	DeadLinks []string
	Subgraphs []string
}

type NoteViews map[string]*NoteView

func (n *NoteView) ID() string {
	return n.Permalink
}

func (n *NoteView) Ast() ast.Node {
	return n.ast
}

func (n *NoteView) SetAst(node ast.Node) {
	n.ast = node
}

func (n *NoteView) ExtractSubgraphs() ([]string, error) {
	subgraphs := make(map[string]struct{})

	err := extractSubgraphs(subgraphs, n.RawMeta["subgraph"])
	if err != nil {
		return nil, fmt.Errorf("error extracting subgraph: %w", err)
	}

	err = extractSubgraphs(subgraphs, n.RawMeta["subgraphs"])
	if err != nil {
		return nil, fmt.Errorf("error extracting subgraphs: %w", err)
	}

	res := make([]string, 0, len(subgraphs))

	for k := range subgraphs {
		res = append(res, k)
	}

	return res, nil
}

func (n *NoteView) ExtractTitle() string {
	title, ok := n.RawMeta["title"]
	if ok {
		str, sOk := title.(string)
		if sOk {
			return str
		}
	}

	// nodeCount := 0
	// docTitle := ""
	//
	// find the first heading in .Ast
	// Need to remove the heading node before rendering
	// ast.Walk(p.Ast, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
	// 	nodeCount++
	//
	// 	if nodeCount > 5 {
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	if n.Kind() == ast.KindHeading {
	// 		heading := n.(*ast.Heading)
	//
	// 		if heading.Level != 1 {
	// 			return ast.WalkContinue, nil
	// 		}
	//
	// 		docTitle = string(heading.Text(p.Content))
	// 		return ast.WalkStop, nil
	// 	}
	//
	// 	return ast.WalkContinue, nil
	// })
	//
	// if docTitle != "" {
	// 	return docTitle
	// }

	return filepath.Base(n.Path[:len(n.Path)-len(".md")])
}

func (pages NoteViews) Subgraphs() ([]string, error) {
	subgraphs := make(map[string]struct{})

	for _, page := range pages {
		for _, ps := range page.Subgraphs {
			subgraphs[ps] = struct{}{}
		}
	}

	res := make([]string, 0, len(subgraphs))

	for k := range subgraphs {
		res = append(res, k)
	}

	return res, nil
}

func extractSubgraphs(target map[string]struct{}, val interface{}) error {
	switch val := val.(type) {
	case string:
		target[val] = struct{}{}
	case []interface{}:
		for _, v := range val {
			if vStr, ok := v.(string); ok {
				target[vStr] = struct{}{}
			} else {
				return fmt.Errorf("invalid subgraph type: %T", v)
			}
		}
	case nil:
		return nil
	default:
		return fmt.Errorf("invalid subgraph type: %T", val)
	}

	return nil
}

func (pages NoteViews) IDMap() map[int64]*NoteView {
	idMap := make(map[int64]*NoteView, len(pages))

	for _, page := range pages {
		idMap[page.PathID] = page
	}

	return idMap
}
