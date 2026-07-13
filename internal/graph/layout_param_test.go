package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/graph/model"
	appmodel "trip2g/internal/model"
)

func TestLayoutBlockParamValue_String(t *testing.T) {
	// Quoted default has quotes stripped.
	v, err := layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "string", Default: `"hello"`})
	require.NoError(t, err)
	sv, ok := v.(*model.StringParamValue)
	require.True(t, ok)
	require.NotNil(t, sv.DefaultValue)
	require.Equal(t, "hello", *sv.DefaultValue)

	// Unquoted default kept as-is.
	v, _ = layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "string", Default: "plain"})
	sv = v.(*model.StringParamValue)
	require.Equal(t, "plain", *sv.DefaultValue)

	// Empty default -> nil default.
	v, _ = layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "string", Default: ""})
	sv = v.(*model.StringParamValue)
	require.Nil(t, sv.DefaultValue)
}

func TestLayoutBlockParamValue_Int(t *testing.T) {
	v, err := layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "int", Default: "42"})
	require.NoError(t, err)
	iv := v.(*model.IntParamValue)
	require.NotNil(t, iv.DefaultValue)
	require.Equal(t, int32(42), *iv.DefaultValue)

	// Unparseable -> nil default (no error).
	v, err = layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "int", Default: "notint"})
	require.NoError(t, err)
	iv = v.(*model.IntParamValue)
	require.Nil(t, iv.DefaultValue)
}

func TestLayoutBlockParamValue_Float(t *testing.T) {
	v, _ := layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "float", Default: "3.14"})
	fv := v.(*model.FloatParamValue)
	require.NotNil(t, fv.DefaultValue)
	require.InDelta(t, 3.14, *fv.DefaultValue, 1e-9)

	v, _ = layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "float", Default: "nan-nope"})
	fv = v.(*model.FloatParamValue)
	require.Nil(t, fv.DefaultValue)
}

func TestLayoutBlockParamValue_Bool(t *testing.T) {
	v, _ := layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "bool", Default: "true"})
	bv := v.(*model.BoolParamValue)
	require.NotNil(t, bv.DefaultValue)
	require.True(t, *bv.DefaultValue)

	v, _ = layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "bool", Default: "false"})
	bv = v.(*model.BoolParamValue)
	require.NotNil(t, bv.DefaultValue)
	require.False(t, *bv.DefaultValue)
}

func TestLayoutBlockParamValue_UnknownType(t *testing.T) {
	v, err := layoutBlockParamValue(&appmodel.LayoutBlockParam{Type: "weird", Default: "x"})
	require.NoError(t, err)
	require.Nil(t, v)
}
