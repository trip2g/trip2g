package layoutloader

import (
	"reflect"
	"regexp"
	"strings"
	"trip2g/internal/model"

	"github.com/CloudyKit/jet/v6"
)

func makeYieldBlocksFunc(blockNames *[]string, warnSink *[]model.NoteWarning) func(jet.Arguments) reflect.Value {
	return func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("yield_blocks", 1, 1)
		var pattern string
		if err := a.ParseInto(&pattern); err != nil {
			return reflect.ValueOf("")
		}
		match, err := compileMatcher(pattern)
		if err != nil {
			*warnSink = append(*warnSink, model.NoteWarning{
				Level:   model.NoteWarningWarning,
				Message: "yield_blocks: invalid pattern \"" + pattern + "\": " + err.Error(),
			})
			return reflect.ValueOf("")
		}
		rt := a.Runtime()
		for _, name := range *blockNames {
			if !match(name) {
				continue
			}
			func() {
				defer func() { _ = recover() }()
				rt.YieldBlock(name, nil)
			}()
		}
		return reflect.ValueOf("")
	}
}

// expandBlockName replaces @lid and @did placeholders derived from sourceID.
// @lid = lodash id (underscores): used for Jet block names.
// @did = dash id (hyphens): used for BEM CSS class names.
// @@lid and @@did are escape sequences that produce literal @lid / @did.
// sourceID examples: "/mesh/bar.html" → @lid="mesh_bar", @did="mesh-bar".
func expandBlockName(content, sourceID string) string {
	if !strings.Contains(content, "@lid") && !strings.Contains(content, "@did") {
		return content
	}
	lid, did := derivePlaceholderIDs(sourceID)

	const sentinelLid = "\x00at_lid\x00"
	const sentinelDid = "\x00at_did\x00"
	content = strings.ReplaceAll(content, "@@lid", sentinelLid)
	content = strings.ReplaceAll(content, "@@did", sentinelDid)
	content = strings.ReplaceAll(content, "@lid", lid)
	content = strings.ReplaceAll(content, "@did", did)
	content = strings.ReplaceAll(content, sentinelLid, "@lid")
	content = strings.ReplaceAll(content, sentinelDid, "@did")
	return content
}

func compileMatcher(p string) (func(string) bool, error) {
	if len(p) >= 2 && p[0] == '/' && p[len(p)-1] == '/' {
		re, err := regexp.Compile(p[1 : len(p)-1])
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}
	return func(name string) bool { return strings.HasPrefix(name, p) }, nil
}
