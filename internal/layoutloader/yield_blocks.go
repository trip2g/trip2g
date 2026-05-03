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
