package codellm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// operatorEnv stands in for codellm's own process environment.
func operatorEnv() []string {
	return []string{"KRISP_TOKEN=a", "KRISP_BASE_URL=b", "GH_TOKEN=c", "UNRELATED=d"}
}

// A role that declares nothing keeps the behaviour that existed before the
// field came back: it sees whatever the operator allowlisted.
func TestEffectiveEnvNames_NoDeclarationKeepsOperatorAllowlist(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(),
		[]string{"KRISP_TOKEN", "GH_TOKEN"}, nil,
		nil, nil)

	require.ElementsMatch(t, []string{"KRISP_TOKEN", "GH_TOKEN"}, got)
}

// Declaring narrows. This is the point of the field: one code role no longer
// sees every secret every other code role on the same codellm needs.
func TestEffectiveEnvNames_DeclarationNarrows(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(),
		[]string{"KRISP_TOKEN", "GH_TOKEN"}, nil,
		[]string{"KRISP_TOKEN"}, nil)

	require.Equal(t, []string{"KRISP_TOKEN"}, got)
}

// A declaration can never reach past the operator's allowlist.
func TestEffectiveEnvNames_DeclarationCannotWiden(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(),
		[]string{"KRISP_TOKEN"}, nil,
		[]string{"KRISP_TOKEN", "GH_TOKEN", "UNRELATED"}, nil)

	require.Equal(t, []string{"KRISP_TOKEN"}, got)
}

func TestEffectiveEnvNames_OperatorPrefix(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(), nil, []string{"KRISP_"}, nil, nil)

	require.ElementsMatch(t, []string{"KRISP_TOKEN", "KRISP_BASE_URL"}, got)
}

// Prefixes on both sides are resolved against the real names, so a role prefix
// and an operator prefix of different lengths still compose correctly.
func TestEffectiveEnvNames_PrefixesOnBothSides(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(), nil, []string{"KR"}, nil, []string{"KRISP_B"})

	require.Equal(t, []string{"KRISP_BASE_URL"}, got)
}

func TestEffectiveEnvNames_RoleNameAndPrefixCombine(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(),
		[]string{"KRISP_TOKEN", "KRISP_BASE_URL", "GH_TOKEN"}, nil,
		[]string{"GH_TOKEN"}, []string{"KRISP_B"})

	require.ElementsMatch(t, []string{"GH_TOKEN", "KRISP_BASE_URL"}, got)
}

// An operator that allowlisted nothing exposes nothing, whatever a role asks.
func TestEffectiveEnvNames_EmptyOperatorAllowlistExposesNothing(t *testing.T) {
	require.Empty(t, effectiveEnvNames(operatorEnv(), nil, nil, []string{"KRISP_TOKEN"}, nil))
}

// A declared name the operator allows but the process does not have is simply
// absent, not an error: the same as before, the code fails when it reads it.
func TestEffectiveEnvNames_AbsentVarIsSkipped(t *testing.T) {
	got := effectiveEnvNames(operatorEnv(), []string{"NOT_SET"}, nil, []string{"NOT_SET"}, nil)

	require.Empty(t, got)
}

func TestEnvDeclarationFromBag(t *testing.T) {
	names, prefixes := envDeclaration(bagJSON(t, map[string]any{
		"env_passthrough": []string{"KRISP_TOKEN"},
		"env_prefix":      []string{"KRISP_"},
	}))

	require.Equal(t, []string{"KRISP_TOKEN"}, names)
	require.Equal(t, []string{"KRISP_"}, prefixes)
}

func TestEnvDeclarationFromBag_AbsentOrMalformed(t *testing.T) {
	names, prefixes := envDeclaration(bagJSON(t, map[string]any{"depth": 1}))
	require.Empty(t, names)
	require.Empty(t, prefixes)

	names, prefixes = envDeclaration([]byte("not json"))
	require.Empty(t, names)
	require.Empty(t, prefixes)
}
