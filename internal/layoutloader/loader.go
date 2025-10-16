package layoutloader

import (
	"fmt"
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
	Content   string
	Assets    map[string]*model.NoteAssetReplace
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
			fmt.Printf("Failed to load layout template: %v nil: %+v\n", err, view == nil)
			// Silently skip templates that fail to load
			delete(jl.templates, source.ID)
			continue
		}

		finder := assetFinder{}
		utils.Walk(view, &finder)

		log.Debug("detect assets", "assets", finder.List)

		assets := []model.LayoutAsset{}

		dir := filepath.Dir(source.Path)

		for _, assetPath := range finder.List {
			assets = append(assets, model.LayoutAsset{
				Path: filepath.Join(dir, assetPath),
			})
		}

		layouts.Map[source.ID] = model.Layout{
			VersionID: source.VersionID,
			Path:      source.Path,
			View:      view,
			Assets:    assets,

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
