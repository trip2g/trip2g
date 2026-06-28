package jsonneteval_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/jsonneteval"
)

func TestNewVM_MaxStack(t *testing.T) {
	vm := jsonneteval.NewVM()
	require.NotNil(t, vm)
	require.Equal(t, 500, vm.MaxStack)
}

func TestEvalJSON_Identity(t *testing.T) {
	src := `std.parseJson(std.extVar("payload"))`
	in := `{"a":1,"b":["x","y"]}`
	out, err := jsonneteval.EvalJSON(src, map[string]string{"payload": in})
	require.NoError(t, err)

	var got, want map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.NoError(t, json.Unmarshal([]byte(in), &want))
	require.Equal(t, want, got)
}

func TestEvalJSON_Remap(t *testing.T) {
	src := `local p = std.parseJson(std.extVar("change")); { renamed: p }`
	out, err := jsonneteval.EvalJSON(src, map[string]string{"change": `[{"path":"a.md"}]`})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.Contains(t, got, "renamed")
}

func TestEvalJSON_RuntimeError(t *testing.T) {
	_, err := jsonneteval.EvalJSON(`error "boom"`, nil)
	require.Error(t, err)
}

func TestValidate(t *testing.T) {
	require.NoError(t, jsonneteval.Validate(`{ ok: std.extVar("payload") }`,
		map[string]string{"payload": "{}"}))
	require.Error(t, jsonneteval.Validate(`}{ not jsonnet`, nil))
}
