package templateviews

import (
	"fmt"
)

// Meta provides access to frontmatter values with type-safe getters.
type Meta struct {
	raw map[string]interface{}
}

// Has returns true if the key exists in frontmatter.
func (m *Meta) Has(key string) bool {
	if m.raw == nil {
		return false
	}
	_, ok := m.raw[key]
	return ok
}

// Get returns the raw value for a key.
func (m *Meta) Get(key string) interface{} {
	if m.raw == nil {
		return nil
	}
	return m.raw[key]
}

// GetString returns a string value or default if not found/wrong type.
func (m *Meta) GetString(key string, def string) string {
	if m.raw == nil {
		return def
	}

	val, ok := m.raw[key]
	if !ok {
		return def
	}

	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return def
	}
}

// GetInt returns an int value or default if not found/wrong type.
func (m *Meta) GetInt(key string, def int) int {
	if m.raw == nil {
		return def
	}

	val, ok := m.raw[key]
	if !ok {
		return def
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return def
	}
}

// GetBool returns a bool value or default if not found/wrong type.
func (m *Meta) GetBool(key string, def bool) bool {
	if m.raw == nil {
		return def
	}

	val, ok := m.raw[key]
	if !ok {
		return def
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "yes" || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return def
	}
}

// Raw returns the underlying raw frontmatter map (for JSON serialization in templates).
func (m *Meta) Raw() map[string]interface{} {
	return m.raw
}

// GetStringSlice returns a string slice or empty slice if not found/wrong type.
func (m *Meta) GetStringSlice(key string) []string {
	if m.raw == nil {
		return nil
	}

	val, ok := m.raw[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			s, isString := item.(string)
			if isString {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{v}
	default:
		return nil
	}
}
