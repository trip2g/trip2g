package model

import "github.com/CloudyKit/jet/v6"

type LayoutAsset struct {
	Path string
	Hash string
}

type Layout struct {
	VersionID int64
	Path      string
	View      *jet.Template `json:"-"`
	Assets    []LayoutAsset
	Content   string `json:"-"` // omit large template content from debug output

	// OriginalContent stores the original file content (JSON for .html.json, HTML for .html)
	// Used for syncing back to clients
	OriginalContent string

	// AssetReplaces is the merged asset URL map shared across all layouts
	// (see Layouts.AssetReplaces). Excluded from JSON to avoid duplication —
	// debug output exposes the shared map once on Layouts.
	AssetReplaces map[string]*NoteAssetReplace `json:"-"`

	// Warnings contains issues detected during loading (e.g., parse errors).
	Warnings []NoteWarning
}

type LayoutSourceFile struct {
	ID        string
	VersionID int64
	Path      string
	Content   string // Processed Jet template content
	Assets    map[string]*NoteAssetReplace

	// OriginalContent stores the original file content before conversion
	// (JSON for .html.json files, same as Content for .html files)
	OriginalContent string
}

type Layouts struct {
	Map    map[string]Layout
	Blocks LayoutBlocks
	// AssetReplaces is the union of all layout-source assets keyed by absolute
	// path. A page (e.g. _layouts/mesh/index.html) yields blocks from component
	// files whose asset() calls must resolve through the page's closure; since
	// note_version_assets links each asset to a single version_id, we merge
	// across all layouts into one shared lookup table.
	AssetReplaces map[string]*NoteAssetReplace
	Load          func(source LayoutSourceFile) Layout `json:"-"`
}

// LayoutBlocks provides block lookup by name or full name.
type LayoutBlocks struct {
	ByName     map[string]LayoutBlock // "header" -> last block with this name
	ByFullName map[string]LayoutBlock // "blocks.html#header" -> block (unique)
}

// Lookup finds a block by name or full name (sourceId#name).
func (lb *LayoutBlocks) Lookup(name string) (LayoutBlock, bool) {
	// First try exact full name match
	if block, ok := lb.ByFullName[name]; ok {
		return block, true
	}

	// Then try by short name
	if block, ok := lb.ByName[name]; ok {
		return block, true
	}

	return LayoutBlock{}, false
}

// All returns all unique blocks (from ByFullName map).
func (lb *LayoutBlocks) All() []LayoutBlock {
	blocks := make([]LayoutBlock, 0, len(lb.ByFullName))
	for _, block := range lb.ByFullName {
		blocks = append(blocks, block)
	}
	return blocks
}

// LayoutBlock represents a block definition found in templates.
type LayoutBlock struct {
	Name       string             // block name (e.g., "header", "cta_section")
	Params     []LayoutBlockParam // parameters with defaults
	HasContent bool               // true if block uses {{ yield content }}
	SourceID   string             // template ID where block is defined
}

// FullName returns unique identifier in format "sourceId#name".
func (b *LayoutBlock) FullName() string {
	return b.SourceID + "#" + b.Name
}

// LayoutBlockParam represents a block parameter.
type LayoutBlockParam struct {
	Name    string // parameter name
	Default string // default value as string
	Type    string // "string" | "int" | "float" | "bool" | "" (unknown)
	Comment string // human-readable description from arg_type
}
