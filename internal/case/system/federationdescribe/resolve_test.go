package federationdescribe_test

import (
	"context"
	"encoding/json"
	"testing"

	"trip2g/internal/case/system/federationdescribe"
	"trip2g/internal/db"

	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/require"
)

type envStub struct {
	federationdescribe.Env
	scope []db.ListFederationSecretScopeByKIDRow
}

func (e *envStub) ListFederationSecretScopeByKID(context.Context, string) ([]db.ListFederationSecretScopeByKIDRow, error) {
	return e.scope, nil
}

// The router serialises a handler's return value only when it marshals itself,
// and answers an empty body otherwise — silently, with a 200. So this asserts
// the answer survives that path rather than that the struct was built.
func TestDescriptionMarshalsThroughTheRouter(t *testing.T) {
	t.Parallel()

	env := &envStub{scope: []db.ListFederationSecretScopeByKIDRow{
		{Name: "office_knowledge", HumanDescription: "door codes and room booking"},
		{Name: "team_status", HumanDescription: ""},
	}}

	description, err := federationdescribe.Resolve(context.Background(), env, "alice")
	require.NoError(t, err)

	marshaler, ok := any(description).(easyjson.Marshaler)
	require.True(t, ok, "the router would answer an empty body for this type")

	raw, err := easyjson.Marshal(marshaler)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	require.EqualValues(t, federationdescribe.Version, got["version"])
	require.Equal(t, "alice", got["kid"])
	require.Equal(t, true, got["rotation"])

	subgraphs, ok := got["subgraphs"].([]any)
	require.True(t, ok)
	require.Len(t, subgraphs, 2)

	first, ok := subgraphs[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "office_knowledge", first["name"])
	require.Equal(t, "door codes and room booking", first["human_description"],
		"the peer is told what the subgraph is; a name on its own is a slug")
}

// A pairing granted nothing authenticates and returns nothing, which from the
// asking side is indistinguishable from a query that matched nothing — unless
// the empty list comes back as an answer rather than as an error.
func TestEmptyScopeIsAnAnswer(t *testing.T) {
	t.Parallel()

	description, err := federationdescribe.Resolve(context.Background(), &envStub{}, "alice")

	require.NoError(t, err)
	require.NotNil(t, description)
	require.Empty(t, description.Subgraphs)

	raw, err := easyjson.Marshal(description)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"subgraphs":[]`)
}
