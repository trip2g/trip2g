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
}

func (jl *jetLoader) Exists(templatePath string) bool {
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

func Load(sourceFiles []SourceFile, options Options) (*model.Layouts, error) {
	jl := &jetLoader{
		templates: make(map[string]string),
	}

	for _, source := range sourceFiles {
		jl.templates[source.ID] = source.Content
	}

	views := jet.NewSet(jl, jet.DevelopmentMode(true))

	views.AddGlobalFunc("asset", func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("asset", 1, 1)
		return reflect.ValueOf("path_to_asset")
	})

	layouts := model.Layouts{
		Map: make(map[string]model.Layout),
	}

	for _, source := range sourceFiles {
		view, err := views.GetTemplate(source.ID)
		if err != nil {
			// Silently skip templates that fail to load
			delete(jl.templates, source.ID)
		}

		finder := assetFinder{}
		utils.Walk(view, &finder)

		assets := []model.LayoutAsset{}

		for _, assetPath := range finder.List {
			assets = append(assets, model.LayoutAsset{
				Path: filepath.Join(options.BasePath, assetPath),
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
	switch node := node.(type) {
	case *jet.IdentifierNode:
		if node.Ident == "asset" {
			w.WaitNext = true
		}

	case *jet.StringNode:
		if w.WaitNext {
			w.List = append(w.List, node.Text)
		}
	}

	vc.Visit(node)

	w.WaitNext = true
}
