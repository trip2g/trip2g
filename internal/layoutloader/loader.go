package layoutloader

import (
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/utils"
)

type SourceFile struct {
	ID        string
	VersionID int64
	Path      string
	Content   string // Processed Jet template content
	Assets    map[string]*model.NoteAssetReplace

	// OriginalContent stores the original file content before conversion
	// (JSON for .html.json files, same as Content for .html files)
	OriginalContent string
}

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

func Load(env Env, sourceFiles []SourceFile, options Options) (*model.Layouts, error) {
	log := logger.WithPrefix(env.Logger(), "layoutloader:")

	jl := &jetLoader{
		templates: make(map[string]string),
		log:       log,
	}

	for _, source := range sourceFiles {
		jl.templates[source.ID] = source.Content
		log.Debug("loaded layout", "id", source.ID)
	}

	layouts := model.Layouts{
		Map: make(map[string]model.Layout),
		Blocks: model.LayoutBlocks{
			ByName: make(map[string][]model.LayoutBlock),
			ByPath: make(map[string]model.LayoutBlock),
		},
	}

	for _, source := range sourceFiles {
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

		view, err := views.GetTemplate(source.ID)
		if err != nil || view == nil {
			// Failed to load layout template - skipping silently
			// Silently skip templates that fail to load
			delete(jl.templates, source.ID)
			continue
		}

		assetWalker := assetFinder{}
		utils.Walk(view, &assetWalker)

		log.Debug("detect assets", "assets", assetWalker.List)

		// Find block definitions
		blockWalker := blockFinder{sourceID: source.ID}
		utils.Walk(view, &blockWalker)

		for _, block := range blockWalker.blocks {
			// Add to ByName (may have duplicates)
			layouts.Blocks.ByName[block.Name] = append(layouts.Blocks.ByName[block.Name], block)
			// Add to ByPath (unique key: sourceID/blockName)
			pathKey := block.SourceID + "/" + block.Name
			layouts.Blocks.ByPath[pathKey] = block
		}

		log.Debug("detect blocks", "count", len(blockWalker.blocks))

		assets := []model.LayoutAsset{}

		dir := filepath.Dir(source.Path)

		for _, assetPath := range assetWalker.List {
			assets = append(assets, model.LayoutAsset{
				Path: filepath.Join(dir, assetPath),
			})
		}

		layouts.Map[source.ID] = model.Layout{
			VersionID:       source.VersionID,
			Path:            source.Path,
			View:            view,
			Assets:          assets,
			Content:         source.Content,
			OriginalContent: source.OriginalContent,

			AssetReplaces: source.Assets,
		}
	}

	return &layouts, nil
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

	// fix the jet panic on missing Parameters
	case *jet.YieldNode:
		node.Parameters = &jet.BlockParameterList{}
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
				}
				info.Params = append(info.Params, param)
			}
		}

		// Check if block has {{ yield content }} inside (in List, not Content)
		info.HasContent = hasYieldContent(block.List)

		w.blocks = append(w.blocks, info)
	}

	vc.Visit(node)
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
