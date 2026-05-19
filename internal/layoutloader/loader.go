package layoutloader

import (
	"fmt"
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
	sourceIDs []string
	log       logger.Logger

	layouts model.Layouts

	pendingWarnings map[string][]model.NoteWarning

	// sets stores the jet.Set per sourceID for the second-pass yield_blocks wiring.
	sets map[string]*jet.Set
	// yieldBlocksSlices stores the mutable block-name slice pointer per sourceID.
	yieldBlocksSlices map[string]*[]string
	// yieldBlocksWarnSinks stores the mutable warn-sink pointer per sourceID.
	yieldBlocksWarnSinks map[string]*[]model.NoteWarning

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
		templates:            make(map[string]string),
		log:                  log,
		pendingWarnings:      make(map[string][]model.NoteWarning),
		sets:                 make(map[string]*jet.Set),
		yieldBlocksSlices:    make(map[string]*[]string),
		yieldBlocksWarnSinks: make(map[string]*[]model.NoteWarning),
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
		jl.templates[source.ID] = expandBlockName(content, source.ID)
		jl.sourceIDs = append(jl.sourceIDs, source.ID)
	}

	// A page (e.g. _layouts/mesh/index.html) yields blocks from component files
	// (cases.html, compat.html, ...). asset() calls inside those components are
	// resolved through the page's closure (source.Assets), but note_version_assets
	// links each SVG to the component file's version_id only — so the page's map
	// is missing them. Asset keys are absolute paths (e.g. _layouts/mesh/x.svg),
	// so a flat union across all layout source files is collision-safe.
	mergedAssets := make(map[string]*model.NoteAssetReplace)
	for _, source := range sourceFiles {
		for k, v := range source.Assets {
			mergedAssets[k] = v
		}
	}
	for i := range sourceFiles {
		sourceFiles[i].Assets = mergedAssets
	}

	jl.layouts = model.Layouts{
		Map: make(map[string]model.Layout),
		Blocks: model.LayoutBlocks{
			ByName:     make(map[string]model.LayoutBlock),
			ByFullName: make(map[string]model.LayoutBlock),
		},
		AssetReplaces: mergedAssets,
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
		LoadPreview: func(main model.LayoutSourceFile, extra map[string]string) (model.Layout, []string) {
			// Snapshot the shared templates map under lock, then release immediately.
			jl.mu.Lock()
			snap := make(map[string]string, len(jl.templates)+len(extra))
			for k, v := range jl.templates {
				snap[k] = v
			}
			jl.mu.Unlock()

			// Caller-supplied files override server files.
			for k, v := range extra {
				snap[k] = v
			}

			// If no inline content provided, fill from snapshot so load() does not
			// overwrite the server template with an empty string.
			if main.Content == "" {
				main.Content = snap[main.ID]
			}

			// Fresh isolated loader — never touches the production jl.
			preview := &jetLoader{
				templates:            snap,
				log:                  jl.log,
				pendingWarnings:      make(map[string][]model.NoteWarning),
				sets:                 make(map[string]*jet.Set),
				yieldBlocksSlices:    make(map[string]*[]string),
				yieldBlocksWarnSinks: make(map[string]*[]model.NoteWarning),
			}

			view, errStr := preview.load(main)
			var warnings []string
			if errStr != "" {
				warnings = append(warnings, "compile: "+errStr)
			}
			if view == nil {
				return model.Layout{}, warnings
			}
			return model.Layout{View: view}, warnings
		},
	}

	jl.processTemplates(sourceFiles)
	jl.wireYieldBlocks(sourceFiles)

	return &jl.layouts, nil
}

// processTemplates is the second pass: load each template, walk for assets and blocks, store layout.
func (jl *jetLoader) processTemplates(sourceFiles []model.LayoutSourceFile) {
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
		// utils.Walk does not recurse into {{ import }} templates — walk them explicitly.
		if views, ok := jl.sets[source.ID]; ok {
			for _, importPath := range extractImportPaths(source.Content) {
				if imported, err := views.GetTemplate(importPath); err == nil {
					utils.Walk(imported, &assetWalker)
				}
			}
		}

		jl.log.Debug("detect assets", "assets", assetWalker.List)

		// Find block definitions
		blockWalker := blockFinder{sourceID: source.ID}
		utils.Walk(view, &blockWalker)
		if views, ok := jl.sets[source.ID]; ok {
			for _, importPath := range extractImportPaths(source.Content) {
				if imported, err := views.GetTemplate(importPath); err == nil {
					utils.Walk(imported, &blockWalker)
				}
			}
		}

		for _, block := range blockWalker.blocks {
			// Add to ByName (last one wins if duplicate names)
			jl.layouts.Blocks.ByName[block.Name] = block
			// Add to ByFullName (unique key: sourceID#blockName)
			jl.layouts.Blocks.ByFullName[block.FullName()] = block
		}

		jl.log.Debug("detect blocks", "count", len(blockWalker.blocks))

		assets := []model.LayoutAsset{}
		seenAsset := make(map[string]struct{})

		dir := filepath.Dir(source.Path)

		for _, assetPath := range assetWalker.List {
			full := filepath.Join(dir, assetPath)
			if _, dup := seenAsset[full]; dup {
				continue
			}
			seenAsset[full] = struct{}{}
			assets = append(assets, model.LayoutAsset{Path: full})
		}

		var warnings []model.NoteWarning
		if pending, ok := jl.pendingWarnings[source.ID]; ok {
			warnings = append(warnings, pending...)
		}

		jl.layouts.Map[source.ID] = model.Layout{
			VersionID:       source.VersionID,
			Path:            source.Path,
			View:            view,
			Assets:          assets,
			Content:         source.Content,
			OriginalContent: source.OriginalContent,

			AssetReplaces: source.Assets,

			Warnings: warnings,
		}
	}
}

// wireYieldBlocks is the third pass: wire yield_blocks for each successfully parsed layout.
func (jl *jetLoader) wireYieldBlocks(sourceFiles []model.LayoutSourceFile) {
	for _, source := range sourceFiles {
		layout, ok := jl.layouts.Map[source.ID]
		if !ok || layout.View == nil {
			continue
		}
		views, hasSet := jl.sets[source.ID]
		blockNamesPtr, hasSlice := jl.yieldBlocksSlices[source.ID]
		if !hasSet || !hasSlice {
			continue
		}

		// Check if this template uses yield_blocks at all; skip if not.
		ybv := &yieldBlocksUsageFinder{}
		utils.Walk(layout.View, ybv)
		if !ybv.found {
			continue
		}

		// Build registry of blocks from all component files (excluding this page).
		registry, regWarnings := buildBlockRegistry(views, jl.sourceIDs, source.ID)

		// Resolve which block names are reachable from this page via yield deps.
		_, inlinedBlockNames, resolveWarnings := resolveNeededFiles(views, layout.View, registry)

		// Populate the slice pointer so the already-registered yield_blocks func sees them.
		*blockNamesPtr = inlinedBlockNames

		// Run validator for static pattern warnings (e.g. invalid regexp).
		validator := &yieldBlocksValidator{}
		utils.Walk(layout.View, validator)

		// Merge all warnings into pendingWarnings so they end up on the layout.
		allWarnings := append(regWarnings, resolveWarnings...) //nolint:gocritic // correct: intentionally creates new combined slice
		allWarnings = append(allWarnings, validator.warnings...)
		if len(allWarnings) > 0 {
			layout.Warnings = append(layout.Warnings, allWarnings...)
			jl.layouts.Map[source.ID] = layout
		}
	}
}

// yieldBlocksUsageFinder checks whether a template contains a yield_blocks call.
type yieldBlocksUsageFinder struct{ found bool }

func (w *yieldBlocksUsageFinder) Visit(vc utils.VisitorContext, node jet.Node) {
	if node == nil {
		return
	}
	if _, ok := node.(*jet.IncludeNode); ok {
		return
	}
	if action, ok := node.(*jet.ActionNode); ok && !w.found {
		if action.Pipe != nil && len(action.Pipe.Cmds) > 0 {
			if ident, identOk := action.Pipe.Cmds[0].BaseExpr.(*jet.IdentifierNode); identOk && ident.Ident == "yield_blocks" {
				w.found = true
				return
			}
		}
	}
	vc.Visit(node)
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
	jl.templates[source.ID] = expandBlockName(content, source.ID)

	// Auto-import: BFS the page's yield calls and inject component definitions
	// via a {{if false}}...{{end}} preamble so that {{ yield component() }} resolves
	// even without an explicit {{ import "component" }} in the page.
	//
	// We do this with a tmpViews set that uses the same loader. The preamble is
	// injected by temporarily rewriting jl.templates[source.ID]; deferred restore
	// ensures the original content is preserved for any subsequent re-reads.
	originalContent := jl.templates[source.ID]
	preambleApplied := false
	defer func() {
		if preambleApplied {
			jl.templates[source.ID] = originalContent
		}
	}()
	if preamble := jl.buildAutoImportPreamble(source.ID); preamble != "" {
		// Jet requires {{ import }} statements at the very top of the template.
		// Insert the preamble AFTER any leading import statements in the original
		// content so we don't break explicit imports the page might already have.
		jl.templates[source.ID] = injectAfterLeadingImports(originalContent, preamble)
		preambleApplied = true
	}

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

	// debug prints the Go type, value, and available methods of any template expression.
	// Usage: {{ debug(note.M()) }} or {{ debug(note.Title) }}
	views.AddGlobalFunc("debug", func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("debug", 1, 1)
		val := a.Get(0)
		if !val.IsValid() {
			return reflect.ValueOf("<nil>")
		}
		iface := val.Interface()
		t := val.Type()
		methods := make([]string, t.NumMethod())
		for i := range methods {
			methods[i] = t.Method(i).Name
		}
		return reflect.ValueOf(fmt.Sprintf("%T: %+v\nmethods: %v", iface, iface, methods))
	})

	// yield_blocks is registered with a mutable slice pointer so the second pass
	// can populate block names after all templates are parsed.
	blockNames := make([]string, 0)
	var warnSink []model.NoteWarning
	views.AddGlobalFunc("yield_blocks", makeYieldBlocksFunc(&blockNames, &warnSink))
	jl.sets[source.ID] = views
	jl.yieldBlocksSlices[source.ID] = &blockNames
	jl.yieldBlocksWarnSinks[source.ID] = &warnSink

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
	if _, ok := node.(*jet.IncludeNode); ok {
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
	if _, ok := node.(*jet.IncludeNode); ok {
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
