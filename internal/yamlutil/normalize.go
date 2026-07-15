// Package yamlutil contains helpers for values produced by YAML decoders.
package yamlutil

import "fmt"

// Normalize converts yaml.v2-style values into JSON-compatible values.
func Normalize(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for key, item := range val {
			m[fmt.Sprint(key)] = Normalize(item)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for key, item := range val {
			m[key] = Normalize(item)
		}
		return m
	case []interface{}:
		items := make([]interface{}, len(val))
		for i, item := range val {
			items[i] = Normalize(item)
		}
		return items
	default:
		return v
	}
}
