package model

import "github.com/CloudyKit/jet/v6"

type LayoutAsset struct {
	Path string
	Hash string
}

type Layout struct {
	VersionID int64
	Path      string
	View      *jet.Template
	Assets    []LayoutAsset

	AssetReplaces map[string]*NoteAssetReplace
}

type Layouts struct {
	Map map[string]Layout
}
