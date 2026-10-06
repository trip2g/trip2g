package layoutloader

import (
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"trip2g/internal/logger"

	"github.com/CloudyKit/jet/v6"
	"gopkg.in/yaml.v3"
)

// jetNil is a nil interface value, the same nil a method returning `any`
// gives Jet, so both `if d` and `d == nil` work on a failed parse.
func jetNil() reflect.Value {
	return reflect.ValueOf((*any)(nil)).Elem()
}

// addParseFuncs registers parseJSON, parseYAML and parseCSV. A parse error
// renders nothing and is logged, so a template can branch on the result:
// {{ if d := parseJSON(b.Content); d }}.
func addParseFuncs(views *jet.Set, log logger.Logger) {
	views.AddGlobalFunc("parseJSON", makeParseFunc("parseJSON", log, func(src string) (any, error) {
		var v any
		err := json.Unmarshal([]byte(src), &v)
		return v, err
	}))

	views.AddGlobalFunc("parseYAML", makeParseFunc("parseYAML", log, func(src string) (any, error) {
		var v any
		err := yaml.Unmarshal([]byte(src), &v)
		return v, err
	}))

	views.AddGlobalFunc("parseCSV", makeParseFunc("parseCSV", log, func(src string) (any, error) {
		reader := csv.NewReader(strings.NewReader(src))
		reader.FieldsPerRecord = -1
		return reader.ReadAll()
	}))
}

func makeParseFunc(name string, log logger.Logger, parse func(string) (any, error)) jet.Func {
	return func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments(name, 1, 1)

		var src string
		err := a.ParseInto(&src)
		if err != nil {
			log.Warn(name+": argument is not a string", "err", err)
			return jetNil()
		}

		v, err := parse(src)
		if err != nil {
			log.Warn(name+": parse failed", "err", err)
			return jetNil()
		}

		if v == nil {
			return jetNil()
		}

		return reflect.ValueOf(v)
	}
}
