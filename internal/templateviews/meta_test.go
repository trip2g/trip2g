package templateviews

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
