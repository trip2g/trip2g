package handletgcanvasupdate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerializeStack_Empty(t *testing.T) {
	require.Equal(t, "[]", serializeStack(nil))
	require.Equal(t, "[]", serializeStack([]string{}))
}

func TestSerializeStack_Values(t *testing.T) {
	got := serializeStack([]string{"a", "b", "c"})
	require.Equal(t, `["a","b","c"]`, got)
}

func TestDeserializeStack_Empty(t *testing.T) {
	require.Nil(t, deserializeStack(""))
	require.Nil(t, deserializeStack("[]"))
}

func TestDeserializeStack_Values(t *testing.T) {
	got := deserializeStack(`["node1","node2"]`)
	require.Equal(t, []string{"node1", "node2"}, got)
}

func TestDeserializeStack_Invalid(t *testing.T) {
	got := deserializeStack("not json")
	require.Nil(t, got)
}

func TestStackRoundtrip(t *testing.T) {
	original := []string{"intro", "pricing", "plans"}
	serialized := serializeStack(original)
	deserialized := deserializeStack(serialized)
	require.Equal(t, original, deserialized)
}
