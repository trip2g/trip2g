package callout

import (
	"strconv"

	"github.com/yuin/goldmark/ast"
)

// Kind is the AST node kind for callout blocks.
var Kind = ast.NewNodeKind("Callout")

// Node represents an Obsidian callout block.
type Node struct {
	ast.BaseBlock
	// CalloutType is the callout type (note, warning, etc.) in lowercase.
	// Named to avoid clashing with ast.Node's Type() NodeType method.
	CalloutType string
	// Title is the optional custom title. If empty, use capitalized CalloutType.
	Title string
	// Foldable is true if the callout has a fold marker (- or +).
	Foldable bool
	// Expanded is true if foldable and starts expanded (+).
	Expanded bool
}

// Kind implements ast.Node.
func (n *Node) Kind() ast.NodeKind {
	return Kind
}

// Dump implements ast.Node.
func (n *Node) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Type":     n.CalloutType,
		"Title":    n.Title,
		"Foldable": strconv.FormatBool(n.Foldable),
		"Expanded": strconv.FormatBool(n.Expanded),
	}, nil)
}
