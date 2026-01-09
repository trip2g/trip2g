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
	Content   string

	AssetReplaces map[string]*NoteAssetReplace
}

type Layouts struct {
	Map    map[string]Layout
	Blocks LayoutBlocks
}

// LayoutBlocks provides block lookup by name or qualified path.
type LayoutBlocks struct {
	ByName map[string][]LayoutBlock // "header" -> [block1, block2...] (may have duplicates)
	ByPath map[string]LayoutBlock   // "blocks/header" -> block (unique)
}

// Lookup finds a block by name or qualified path.
// Returns the block, whether it was found, and an error message if ambiguous.
func (lb *LayoutBlocks) Lookup(name string) (LayoutBlock, bool, string) {
	// First try exact path match
	if block, ok := lb.ByPath[name]; ok {
		return block, true, ""
	}

	// Then try by name
	blocks, ok := lb.ByName[name]
	if !ok || len(blocks) == 0 {
		return LayoutBlock{}, false, ""
	}

	if len(blocks) == 1 {
		return blocks[0], true, ""
	}

	// Ambiguous - multiple blocks with same name
	paths := make([]string, len(blocks))
	for i, b := range blocks {
		paths[i] = b.SourceID + "/" + b.Name
	}
	return LayoutBlock{}, false, "block '" + name + "' is ambiguous, use: " + joinPaths(paths)
}

func joinPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	result := "'" + paths[0] + "'"
	for i := 1; i < len(paths); i++ {
		result += " or '" + paths[i] + "'"
	}
	return result
}

// LayoutBlock represents a block definition found in templates.
type LayoutBlock struct {
	Name       string             // block name (e.g., "header", "cta_section")
	Params     []LayoutBlockParam // parameters with defaults
	HasContent bool               // true if block uses {{ yield content }}
	SourceID   string             // template ID where block is defined
}

// LayoutBlockParam represents a block parameter.
type LayoutBlockParam struct {
	Name    string // parameter name
	Default string // default value as string
}
