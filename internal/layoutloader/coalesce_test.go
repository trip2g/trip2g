package layoutloader

import (
	"bytes"
	"reflect"
	"testing"
	"trip2g/internal/logger"
	"trip2g/internal/model"

	"github.com/stretchr/testify/require"
)

func TestCoalesceRender(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"first arg valid returned", `{{ coalesce("a", "b") }}`, "a"},
		{"first empty skipped, second returned", `{{ coalesce("", "b") }}`, "b"},
		{"all empty falls back to last", `{{ coalesce("", "") }}`, ""},
		{"map hit returned", `{{ m := map("en", "x", "ru", "y") }}{{ coalesce(m["ru"], m["en"]) }}`, "y"},
		{"map miss falls back", `{{ m := map("en", "x") }}{{ coalesce(m["ru"], m["en"]) }}`, "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := []model.LayoutSourceFile{{
				ID:      "/page",
				Path:    "_layouts/page.html",
				Content: tt.content,
			}}

			layouts, err := Load(&testEnv{logger: &logger.TestLogger{}}, sources, Options{})
			require.NoError(t, err)

			var buf bytes.Buffer
			err = layouts.Map["/page"].View.Execute(&buf, nil, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, buf.String())
		})
	}
}

func TestIsEmptyValue(t *testing.T) {
	var nilPtr *string
	s := "x"

	tests := []struct {
		name string
		v    reflect.Value
		want bool
	}{
		{"invalid value", reflect.Value{}, true},
		{"empty string", reflect.ValueOf(""), true},
		{"non-empty string", reflect.ValueOf("x"), false},
		{"nil pointer", reflect.ValueOf(nilPtr), true},
		{"non-nil pointer", reflect.ValueOf(&s), false},
		{"empty slice", reflect.ValueOf([]string{}), true},
		{"non-empty slice", reflect.ValueOf([]string{"a"}), false},
		{"empty map", reflect.ValueOf(map[string]string{}), true},
		{"non-empty map", reflect.ValueOf(map[string]string{"a": "b"}), false},
		{"nil interface", reflect.ValueOf([]interface{}{nil}).Index(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isEmptyValue(tt.v))
		})
	}
}
