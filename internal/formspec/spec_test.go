package formspec_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trip2g/internal/formspec"
)

func TestParseFromRawMeta_nil_when_no_form_key(t *testing.T) {
	spec, err := formspec.ParseFromRawMeta(map[string]interface{}{
		"title": "Hello",
	})
	require.NoError(t, err)
	require.Nil(t, spec)
}

func TestParseFromRawMeta_basic_fields(t *testing.T) {
	rawMeta := map[string]interface{}{
		"form": map[string]interface{}{
			"can_submit": "guest",
			"fields": []interface{}{
				map[string]interface{}{
					"name":     "email",
					"type":     "email",
					"required": true,
				},
				map[string]interface{}{
					"name": "rating",
					"type": "int",
					"min":  1,
					"max":  5,
				},
			},
		},
	}
	spec, err := formspec.ParseFromRawMeta(rawMeta)
	require.NoError(t, err)
	require.NotNil(t, spec)
	require.Equal(t, formspec.CanSubmitGuest, spec.CanSubmit)
	require.Len(t, spec.Fields, 2)
	require.Equal(t, "email", spec.Fields[0].Name)
	require.True(t, spec.Fields[0].Required)
	require.Equal(t, formspec.FieldTypeInt, spec.Fields[1].Type)
}

func TestParseFromRawMeta_enum_validation(t *testing.T) {
	rawMeta := map[string]interface{}{
		"form": map[string]interface{}{
			"can_submit": "admin",
			"fields": []interface{}{
				map[string]interface{}{
					"name": "deadline",
					"type": "text",
					"enum": []interface{}{"yesterday", "month", "quarter"},
				},
			},
		},
	}
	spec, err := formspec.ParseFromRawMeta(rawMeta)
	require.NoError(t, err)
	require.NotNil(t, spec)
	require.Equal(t, []string{"yesterday", "month", "quarter"}, spec.Fields[0].StringEnum)
}

func TestParseFormRef_wikilink(t *testing.T) {
	kind, value, ok := formspec.ParseFormRef(map[string]interface{}{
		"form_ref": "[[Comment Form]]",
	})
	require.True(t, ok)
	require.Equal(t, "wikilink", kind)
	require.Equal(t, "Comment Form", value)
}

func TestParseFormRef_path(t *testing.T) {
	kind, value, ok := formspec.ParseFormRef(map[string]interface{}{
		"form_ref": "templates/comment-form.md",
	})
	require.True(t, ok)
	require.Equal(t, "path", kind)
	require.Equal(t, "templates/comment-form.md", value)
}

func TestParseFormRef_absent(t *testing.T) {
	_, _, ok := formspec.ParseFormRef(map[string]interface{}{})
	require.False(t, ok)
}
