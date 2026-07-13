package graph

import (
	"strconv"

	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

// layoutBlockParamValue coerces a layout block parameter's string default into
// the typed GraphQL union value. Unknown types and unparseable defaults yield a
// nil default (or nil value for unknown types) — presentation-only, no errors.
func layoutBlockParamValue(obj *appmodel.LayoutBlockParam) (model.LayoutBlockParamValue, error) {
	switch obj.Type {
	case "string":
		var defaultVal *string
		if obj.Default != "" {
			// Remove quotes from string default (e.g., `"hello"` -> `hello`)
			s := obj.Default
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				s = s[1 : len(s)-1]
			}
			defaultVal = &s
		}
		return &model.StringParamValue{DefaultValue: defaultVal}, nil

	case "int":
		var defaultVal *int32
		if obj.Default != "" {
			v, err := strconv.ParseInt(obj.Default, 10, 32)
			if err == nil {
				i32 := int32(v)
				defaultVal = &i32
			}
		}
		return &model.IntParamValue{DefaultValue: defaultVal}, nil

	case "float":
		var defaultVal *float64
		if obj.Default != "" {
			v, err := strconv.ParseFloat(obj.Default, 64)
			if err == nil {
				defaultVal = &v
			}
		}
		return &model.FloatParamValue{DefaultValue: defaultVal}, nil

	case "bool":
		var defaultVal *bool
		if obj.Default != "" {
			v := obj.Default == "true"
			defaultVal = &v
		}
		return &model.BoolParamValue{DefaultValue: defaultVal}, nil

	default:
		// Unknown type - return nil (no value info)
		return nil, nil
	}
}
