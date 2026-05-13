package formspec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type CanSubmit string

const (
	CanSubmitGuest    CanSubmit = "guest"
	CanSubmitPaidUser CanSubmit = "paid_user"
	CanSubmitAdmin    CanSubmit = "admin"
)

type FieldType string

const (
	FieldTypeText  FieldType = "text"
	FieldTypeEmail FieldType = "email"
	FieldTypeInt   FieldType = "int"
	FieldTypeBool  FieldType = "bool"
	FieldTypeFile  FieldType = "file" // v2
)

type FormField struct {
	Name       string    `yaml:"name"       json:"name"`
	Type       FieldType `yaml:"type"       json:"type"`
	Required   bool      `yaml:"required"   json:"required"`
	MinLength  *int      `yaml:"min_length" json:"min_length,omitempty"`
	MaxLength  *int      `yaml:"max_length" json:"max_length,omitempty"`
	Min        *int      `yaml:"min"        json:"min,omitempty"`
	Max        *int      `yaml:"max"        json:"max,omitempty"`
	StringEnum []string  `yaml:"-"          json:"string_enum,omitempty"`
	IntEnum    []int     `yaml:"-"          json:"int_enum,omitempty"`
	BoolEnum   []bool    `yaml:"-"          json:"bool_enum,omitempty"`
}

type FormSpec struct {
	CanSubmit CanSubmit   `yaml:"can_submit" json:"can_submit"`
	Turnstile bool        `yaml:"turnstile"  json:"turnstile"`
	Fields    []FormField `yaml:"fields"     json:"fields"`
}

// ParseFromRawMeta extracts FormSpec from a note's RawMeta.
// Returns nil, nil if no form key is present.
func ParseFromRawMeta(rawMeta map[string]interface{}) (*FormSpec, error) {
	formRaw, ok := rawMeta["form"]
	if !ok {
		return nil, nil
	}
	b, err := yaml.Marshal(formRaw)
	if err != nil {
		return nil, fmt.Errorf("formspec: marshal: %w", err)
	}
	type rawField struct {
		Name      string      `yaml:"name"`
		Type      FieldType   `yaml:"type"`
		Required  bool        `yaml:"required"`
		MinLength *int        `yaml:"min_length"`
		MaxLength *int        `yaml:"max_length"`
		Min       *int        `yaml:"min"`
		Max       *int        `yaml:"max"`
		Enum      interface{} `yaml:"enum"`
	}
	type rawSpec struct {
		CanSubmit CanSubmit  `yaml:"can_submit"`
		Turnstile bool       `yaml:"turnstile"`
		Fields    []rawField `yaml:"fields"`
	}
	var raw rawSpec
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("formspec: unmarshal: %w", err)
	}
	spec := &FormSpec{
		CanSubmit: raw.CanSubmit,
		Turnstile: raw.Turnstile,
		Fields:    make([]FormField, len(raw.Fields)),
	}
	for i, rf := range raw.Fields {
		f := FormField{
			Name:      rf.Name,
			Type:      rf.Type,
			Required:  rf.Required,
			MinLength: rf.MinLength,
			MaxLength: rf.MaxLength,
			Min:       rf.Min,
			Max:       rf.Max,
		}
		f.StringEnum, f.IntEnum, f.BoolEnum = parseEnum(rf.Enum)
		spec.Fields[i] = f
	}
	return spec, nil
}

func parseEnum(raw interface{}) (strs []string, ints []int, bools []bool) {
	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 {
		return nil, nil, nil
	}
	switch items[0].(type) {
	case string:
		for _, v := range items {
			if s, ok := v.(string); ok {
				strs = append(strs, s)
			}
		}
	case int, int64, float64:
		for _, v := range items {
			switch n := v.(type) {
			case int:
				ints = append(ints, n)
			case int64:
				ints = append(ints, int(n))
			case float64:
				ints = append(ints, int(n))
			}
		}
	case bool:
		for _, v := range items {
			if b, ok := v.(bool); ok {
				bools = append(bools, b)
			}
		}
	}
	return
}

// ParseFormRef extracts form_ref from rawMeta.
// Returns kind ("wikilink" or "path"), value, and ok.
func ParseFormRef(rawMeta map[string]interface{}) (kind, value string, ok bool) {
	refRaw, exists := rawMeta["form_ref"]
	if !exists {
		return "", "", false
	}
	s, isStr := refRaw.(string)
	if !isStr {
		return "", "", false
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[[") && strings.HasSuffix(s, "]]") {
		title := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "[["), "]]"))
		return "wikilink", title, true
	}
	return "path", s, true
}
