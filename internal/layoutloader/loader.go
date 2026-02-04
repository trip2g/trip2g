package layoutloader

import (
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

type Loader struct {
}

type Env interface {
	Logger() logger.Logger
}

// func New(env Env) *Loader {
// 	return &Loader{}
// }

type jetLoader struct {
	templates map[string]string
	log       logger.Logger

	layouts model.Layouts

	mu sync.Mutex
}

func (jl *jetLoader) Exists(templatePath string) bool {
	jl.log.Debug("checking existence of template", "path", templatePath)
	_, exists := jl.templates[templatePath]
	return exists
}

func (jl *jetLoader) Open(templatePath string) (io.ReadCloser, error) {
	content := jl.templates[templatePath]
	return io.NopCloser(strings.NewReader(content)), nil
}

type Options struct {
	BasePath string
}

func Load(env Env, sourceFiles []model.LayoutSourceFile, options Options) (*model.Layouts, error) {
	log := logger.WithPrefix(env.Logger(), "layoutloader:")

	jl := &jetLoader{
		templates: make(map[string]string),
		log:       log,
	}

	// First pass: add all templates to map (for cross-imports)
	for _, source := range sourceFiles {
		content := source.Content
		if strings.HasSuffix(source.Path, ".html.json") {
			jetContent, err := ConvertJSONLayout([]byte(content))
			if err != nil {
				log.Error("failed to convert json layout", "path", source.Path, "error", err)
				continue
			}
			content = jetContent
		}
		jl.templates[source.ID] = content
	}

	jl.layouts = model.Layouts{
		Map: make(map[string]model.Layout),
		Blocks: model.LayoutBlocks{
			ByName:     make(map[string]model.LayoutBlock),
			ByFullName: make(map[string]model.LayoutBlock),
		},
		Load: func(source model.LayoutSourceFile) model.Layout {
			view, parseErr := jl.load(source)
			layout := model.Layout{View: view}
			if parseErr != "" {
				layout.Warnings = []model.NoteWarning{{
					Level:   model.NoteWarningCritical,
					Message: parseErr,
				}}
			}
			return layout
		},
	}

	for _, source := range sourceFiles {
		view, parseErr := jl.load(source)
		if parseErr != "" {
			// Store layout with parse error for fallback rendering
			jl.layouts.Map[source.ID] = model.Layout{
				VersionID:       source.VersionID,
				Path:            source.Path,
				View:            nil,
				Content:         source.Content,
				OriginalContent: source.OriginalContent,
				AssetReplaces:   source.Assets,
				Warnings: []model.NoteWarning{{
					Level:   model.NoteWarningCritical,
					Message: parseErr,
				}},
			}
			delete(jl.templates, source.ID)
			continue
		}

		assetWalker := assetFinder{}
		utils.Walk(view, &assetWalker)

		jl.log.Debug("detect assets", "assets", assetWalker.List)

		// Find block definitions
		blockWalker := blockFinder{sourceID: source.ID}
		utils.Walk(view, &blockWalker)

		for _, block := range blockWalker.blocks {
			// Add to ByName (last one wins if duplicate names)
			jl.layouts.Blocks.ByName[block.Name] = block
			// Add to ByFullName (unique key: sourceID#blockName)
			jl.layouts.Blocks.ByFullName[block.FullName()] = block
		}

		jl.log.Debug("detect blocks", "count", len(blockWalker.blocks))

		assets := []model.LayoutAsset{}

		dir := filepath.Dir(source.Path)

		for _, assetPath := range assetWalker.List {
			assets = append(assets, model.LayoutAsset{
				Path: filepath.Join(dir, assetPath),
			})
		}

		jl.layouts.Map[source.ID] = model.Layout{
			VersionID:       source.VersionID,
			Path:            source.Path,
			View:            view,
			Assets:          assets,
			Content:         source.Content,
			OriginalContent: source.OriginalContent,

			AssetReplaces: source.Assets,
		}
	}

	return &jl.layouts, nil
}

// load parses a template and returns (template, parseError).
// If parsing fails, returns (nil, errorMessage).
func (jl *jetLoader) load(source model.LayoutSourceFile) (*jet.Template, string) {
	jl.mu.Lock()
	defer jl.mu.Unlock()

	// Add content to templates map (auto-convert JSON if needed)
	content := source.Content
	if strings.HasSuffix(source.Path, ".html.json") {
		jetContent, err := ConvertJSONLayout([]byte(content))
		if err != nil {
			jl.log.Error("failed to convert json layout", "path", source.Path, "error", err)
			return nil, err.Error()
		}
		content = jetContent
	}
	jl.templates[source.ID] = content

	views := jet.NewSet(jl, jet.DevelopmentMode(true))

	sourceDir := filepath.Dir(source.Path)

	views.AddGlobalFunc("asset", func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("asset", 1, 1)

		var val string

		err := a.ParseInto(&val)
		if err != nil {
			return reflect.ValueOf("invalid value")
		}

		key := filepath.Join(sourceDir, val)

		asset, exists := source.Assets[key]
		if exists {
			return reflect.ValueOf(asset.URL)
		}

		return reflect.ValueOf(val)
	})

	// arg_type is a metadata directive for block parameters.
	// Runtime: returns empty string (no effect on rendering).
	// Parse time: blockFinder extracts type/comment info from AST.
	// Usage: {{ arg_type "paramName" "type" "description" }}
	views.AddGlobalFunc("arg_type", func(a jet.Arguments) reflect.Value {
		return reflect.ValueOf("")
	})

	view, err := views.GetTemplate(source.ID)
	if err != nil || view == nil {
		errMsg := "unknown error"
		if err != nil {
			errMsg = err.Error()
		}
		jl.log.Error("failed to load template", "id", source.ID, "err", err)
		return nil, errMsg
	}

	return view, ""
}

type assetFinder struct {
	List     []string
	WaitNext bool
}

func (w *assetFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}

	switch node := node.(type) {
	case *jet.IdentifierNode:
		if node.Ident == "asset" {
			w.WaitNext = true
			vc.Visit(node)
			return
		}

	case *jet.StringNode:
		if w.WaitNext {
			w.List = append(w.List, node.Text)
		}

	// fix the jet panic on missing Parameters (only if nil, don't clear existing params!)
	case *jet.YieldNode:
		if node.Parameters == nil {
			node.Parameters = &jet.BlockParameterList{}
		}
	}

	vc.Visit(node)

	w.WaitNext = false
}

// blockFinder walks the AST to find block definitions.
type blockFinder struct {
	sourceID string
	blocks   []model.LayoutBlock
}

func (w *blockFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}

	if block, ok := node.(*jet.BlockNode); ok {
		info := model.LayoutBlock{
			Name:     block.Name,
			SourceID: w.sourceID,
		}

		// Extract parameters
		if block.Parameters != nil {
			for _, p := range block.Parameters.List {
				param := model.LayoutBlockParam{
					Name: p.Identifier,
				}
				if p.Expression != nil {
					param.Default = p.Expression.String()
					param.Type = inferTypeFromExpr(p.Expression)
				}
				info.Params = append(info.Params, param)
			}
		}

		// Extract arg_type metadata from block content
		argTypes := extractArgTypes(block.List)
		for i := range info.Params {
			if meta, found := argTypes[info.Params[i].Name]; found {
				info.Params[i].Type = meta.Type
				info.Params[i].Comment = meta.Comment
			}
		}

		// Check if block has {{ yield content }} inside (in List, not Content)
		info.HasContent = hasYieldContent(block.List)

		w.blocks = append(w.blocks, info)
	}

	vc.Visit(node)
}

// argTypeMeta holds metadata extracted from arg_type directive.
type argTypeMeta struct {
	Type    string
	Comment string
}

// extractArgTypes finds all arg_type calls in block content.
// Returns map of param name -> metadata.
func extractArgTypes(node jet.Node) map[string]argTypeMeta {
	result := make(map[string]argTypeMeta)
	extractArgTypesRecursive(node, result)
	return result
}

func extractArgTypesRecursive(node jet.Node, result map[string]argTypeMeta) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *jet.ListNode:
		if n != nil {
			for _, child := range n.Nodes {
				extractArgTypesRecursive(child, result)
			}
		}
	case *jet.ActionNode:
		// ActionNode contains a PipeNode
		if n.Pipe != nil {
			extractArgTypesFromPipe(n.Pipe, result)
		}
	}
}

func extractArgTypesFromPipe(pipe *jet.PipeNode, result map[string]argTypeMeta) {
	if pipe == nil || len(pipe.Cmds) == 0 {
		return
	}

	cmd := pipe.Cmds[0]

	// BaseExpr should be identifier "arg_type"
	ident, ok := cmd.BaseExpr.(*jet.IdentifierNode)
	if !ok || ident.Ident != "arg_type" {
		return
	}

	// Need at least 2 args: name and type
	if len(cmd.Exprs) < 2 {
		return
	}

	// Extract param name (1st arg)
	nameNode, ok := cmd.Exprs[0].(*jet.StringNode)
	if !ok {
		return
	}

	// Extract type (2nd arg)
	typeNode, ok := cmd.Exprs[1].(*jet.StringNode)
	if !ok {
		return
	}

	meta := argTypeMeta{
		Type: typeNode.Text,
	}

	// Extract comment (3rd arg, optional)
	if len(cmd.Exprs) >= 3 {
		if commentNode, isStr := cmd.Exprs[2].(*jet.StringNode); isStr {
			meta.Comment = commentNode.Text
		}
	}

	result[nameNode.Text] = meta
}

// inferTypeFromExpr determines param type from default value AST node.
func inferTypeFromExpr(expr jet.Expression) string {
	switch e := expr.(type) {
	case *jet.StringNode:
		return "string"
	case *jet.NumberNode:
		if e.IsInt || e.IsUint {
			return "int"
		}
		return "float"
	case *jet.BoolNode:
		return "bool"
	case *jet.NilNode:
		return "nil"
	default:
		return ""
	}
}

// hasYieldContent recursively checks if a block contains {{ yield content }}.
func hasYieldContent(node jet.Node) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case *jet.YieldNode:
		if n.IsContent {
			return true
		}
	case *jet.BlockNode:
		if hasYieldContent(n.List) || hasYieldContent(n.Content) {
			return true
		}
	case *jet.ListNode:
		if n != nil {
			for _, child := range n.Nodes {
				if hasYieldContent(child) {
					return true
				}
			}
		}
	case *jet.IfNode:
		if hasYieldContent(n.List) || hasYieldContent(n.ElseList) {
			return true
		}
	case *jet.RangeNode:
		if hasYieldContent(n.List) || hasYieldContent(n.ElseList) {
			return true
		}
	}

	return false
}
