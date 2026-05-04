package layoutloader

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/CloudyKit/jet/v6"
)

// leadingImportRegexp matches `{{ import "..." }}` (with optional whitespace and
// surrounding blank lines). Jet requires imports at the start of the template.
var leadingImportRegexp = regexp.MustCompile(`^\s*{{\s*import\s+"[^"]*"\s*}}\s*`)

// importPathRegexp matches any `{{ import "path" }}` in a template, not just leading ones.
var importPathRegexp = regexp.MustCompile(`{{\s*import\s+"([^"]+)"\s*}}`)

// extractImportPaths returns all import paths found anywhere in content.
func extractImportPaths(content string) []string {
	matches := importPathRegexp.FindAllStringSubmatch(content, -1)
	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		paths = append(paths, m[1])
	}
	return paths
}

// injectAfterLeadingImports returns content with preamble inserted after any
// leading {{ import "..." }} statements. If there are no leading imports, the
// preamble is prepended to the content.
func injectAfterLeadingImports(content, preamble string) string {
	cut := 0
	for {
		loc := leadingImportRegexp.FindStringIndex(content[cut:])
		if loc == nil {
			break
		}
		cut += loc[1]
	}
	return content[:cut] + preamble + content[cut:]
}

// buildAutoImportPreamble walks the yield-dependency graph starting from the page
// at sourceID and returns a `{{if false}}...{{end}}` preamble containing the raw
// content of every component file whose blocks are reachable via {{ yield X() }}.
//
// The preamble lets Jet's parser register the component blocks in processedBlocks
// (so {{ yield X() }} resolves) without rendering them inline. This mirrors what
// an explicit `{{ import "/comp" }}` would do, but is computed automatically by
// scanning what the page actually yields.
//
// Returns "" when no components are needed (e.g. a self-contained page).
//
// IMPORTANT: callers must already hold jl.mu — this function reads jl.templates.
func (jl *jetLoader) buildAutoImportPreamble(pageSourceID string) string {
	// Use a separate jet.Set to parse the page and component templates without
	// polluting the page's real Set.
	tmpViews := jet.NewSet(jl, jet.DevelopmentMode(true))
	// `arg_type` and `yield_blocks` are referenced inside component bodies; provide
	// no-op implementations so that GetTemplate parses successfully.
	tmpViews.AddGlobalFunc("arg_type", func(a jet.Arguments) reflect.Value {
		return reflect.ValueOf("")
	})
	tmpViews.AddGlobalFunc("yield_blocks", func(a jet.Arguments) reflect.Value {
		return reflect.ValueOf("")
	})
	tmpViews.AddGlobalFunc("asset", func(a jet.Arguments) reflect.Value {
		return reflect.ValueOf("")
	})

	pageView, err := tmpViews.GetTemplate(pageSourceID)
	if err != nil || pageView == nil {
		// If page parsing fails here we let the main load() path produce the error;
		// no preamble in this case.
		return ""
	}

	registry, _ := buildBlockRegistry(tmpViews, jl.sourceIDs, pageSourceID)
	neededFileIDs, _, _ := resolveNeededFiles(tmpViews, pageView, registry)

	if len(neededFileIDs) == 0 {
		return ""
	}

	var pre strings.Builder
	pre.WriteString("{{if false}}\n")
	for _, fid := range neededFileIDs {
		// Strip leading {{ import "..." }} statements from component bodies.
		// Inside the {{if false}}...{{end}} preamble, imports are no longer at
		// the top of the template, which Jet rejects. Components are inlined
		// transitively via the BFS in resolveNeededFiles, so dropping their own
		// imports is safe.
		pre.WriteString(stripLeadingImports(jl.templates[fid]))
		pre.WriteByte('\n')
	}
	pre.WriteString("{{end}}\n")
	return pre.String()
}

// stripLeadingImports removes any number of leading {{ import "..." }} statements
// from content (with optional surrounding whitespace).
func stripLeadingImports(content string) string {
	for {
		loc := leadingImportRegexp.FindStringIndex(content)
		if loc == nil {
			return content
		}
		content = content[loc[1]:]
	}
}
