package templateviews

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMeta_Debug(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		want string
	}{
		{
			name: "nil raw returns empty object",
			raw:  nil,
			want: "{}",
		},
		{
			name: "string value",
			raw:  map[string]interface{}{"title": "My Page"},
			want: `{"title":"My Page"}`,
		},
		{
			name: "list value",
			raw:  map[string]interface{}{"tags": []interface{}{"a", "b"}},
			want: `{"tags":["a","b"]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Meta{raw: tc.raw}
			require.Equal(t, tc.want, m.Debug())
		})
	}
}

func TestMeta_GetStrings(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		key  string
		want []string
	}{
		{
			name: "nil raw map returns empty slice",
			raw:  nil,
			key:  "extra_content",
			want: []string{},
		},
		{
			name: "missing key returns empty slice",
			raw:  map[string]interface{}{},
			key:  "extra_content",
			want: []string{},
		},
		{
			name: "yaml list ([]interface{}) returns string slice",
			raw:  map[string]interface{}{"extra_content": []interface{}{"channels", "prices", "faq"}},
			key:  "extra_content",
			want: []string{"channels", "prices", "faq"},
		},
		{
			name: "[]string returns as-is",
			raw:  map[string]interface{}{"extra_content": []string{"a", "b"}},
			key:  "extra_content",
			want: []string{"a", "b"},
		},
		{
			name: "single string wrapped in slice",
			raw:  map[string]interface{}{"extra_content": "channels"},
			key:  "extra_content",
			want: []string{"channels"},
		},
		{
			name: "wrong type returns empty slice",
			raw:  map[string]interface{}{"extra_content": 42},
			key:  "extra_content",
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Meta{raw: tc.raw}
			got := m.GetStrings(tc.key)
			require.NotNil(t, got, "GetStrings must never return nil (breaks Jet range)")
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMeta_GetInt(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want int
	}{
		{name: "int", val: 3, want: 3},
		{name: "float is truncated", val: 2.7, want: 2},
		{name: "numeric string", val: "3", want: 3},
		{name: "numeric string with spaces", val: " -4 ", want: -4},
		{name: "non-numeric string", val: "three", want: 7},
		{name: "float string", val: "3.5", want: 7},
		{name: "bool", val: true, want: 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Meta{raw: map[string]interface{}{"n": tc.val}}
			require.Equal(t, tc.want, m.GetInt("n", 7))
		})
	}
}

func TestMeta_GetFloat(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want float64
	}{
		{name: "float", val: 9.99, want: 9.99},
		{name: "int", val: 3, want: 3},
		{name: "int64", val: int64(4), want: 4},
		{name: "numeric string", val: " 9.99 ", want: 9.99},
		{name: "non-numeric string", val: "cheap", want: 7},
		{name: "NaN string", val: "NaN", want: 7},
		{name: "bool", val: true, want: 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Meta{raw: map[string]interface{}{"n": tc.val}}
			require.InDelta(t, tc.want, m.GetFloat("n", 7), 1e-9)
		})
	}
}
