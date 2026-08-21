package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/fleetinput"
)

func decodeBag(t *testing.T, raw []byte) fleetinput.Input {
	t.Helper()
	var bag fleetinput.Input
	require.NoError(t, json.Unmarshal(raw, &bag))
	return bag
}

// A role's own frontmatter travels in the bag, so code can read its
// configuration from the note that declares it instead of hardcoding it in the
// body.
func TestBuildInputBag_CarriesRoleFrontmatter(t *testing.T) {
	role, err := ParseRole("roles/ingest.md", "body", map[string]string{
		"fleet_id":       "codellm",
		"mode":           "cron",
		"cron_schedule":  "* * * * *",
		"krisp_base_url": "https://api.krisp.ai",
	})
	require.NoError(t, err)

	bag := decodeBag(t, buildInputBag(role, renderCtx{Depth: 1}))

	require.Equal(t, "https://api.krisp.ai", bag.Frontmatter["krisp_base_url"])
	require.Equal(t, "codellm", bag.Frontmatter["fleet_id"])
	require.Equal(t, 1, bag.Depth)
}

func TestBuildCronInputBag_CarriesRoleFrontmatter(t *testing.T) {
	role, err := ParseRole("roles/ingest.md", "body", map[string]string{
		"fleet_id":       "codellm",
		"krisp_base_url": "https://api.krisp.ai",
	})
	require.NoError(t, err)

	bag := decodeBag(t, buildCronInputBag(role, renderCtx{}))

	require.Equal(t, "https://api.krisp.ai", bag.Frontmatter["krisp_base_url"])
}

// A role with no frontmatter must not produce a null map the other side has to
// special-case.
func TestBuildInputBag_EmptyFrontmatterIsAnEmptyMap(t *testing.T) {
	role, err := ParseRole("roles/x.md", "body", map[string]string{})
	require.NoError(t, err)

	bag := decodeBag(t, buildInputBag(role, renderCtx{}))

	require.NotNil(t, bag.Frontmatter)
	require.Empty(t, bag.Frontmatter)
}

// ParseRole keeps the raw frontmatter, and keeps its OWN copy: discovery reuses
// the map it builds per note, and a later mutation there must not reach into a
// parsed Role.
func TestParseRole_RetainsFrontmatterCopy(t *testing.T) {
	meta := map[string]string{"fleet_id": "codellm", "custom": "one"}

	role, err := ParseRole("roles/x.md", "body", meta)
	require.NoError(t, err)

	meta["custom"] = "two"
	require.Equal(t, "one", role.Frontmatter["custom"])
}

// A role narrows which of codellm's allowlisted env vars its code receives.
// fleet only forwards the declaration; codellm intersects it with the operator
// allowlist, so this can never widen access.
func TestBuildInputBag_CarriesEnvDeclaration(t *testing.T) {
	role, err := ParseRole("roles/ingest.md", "body", map[string]string{
		"fleet_id":        "codellm",
		"env_passthrough": "[KRISP_TOKEN, KRISP_BASE_URL]",
		"env_prefix":      "[KRISP_]",
	})
	require.NoError(t, err)

	bag := decodeBag(t, buildInputBag(role, renderCtx{}))

	require.Equal(t, []string{"KRISP_TOKEN", "KRISP_BASE_URL"}, bag.EnvPassthrough)
	require.Equal(t, []string{"KRISP_"}, bag.EnvPrefix)
}

// A role that declares neither field keeps the pre-existing behaviour, so notes
// written before the fields came back are unaffected.
func TestBuildInputBag_NoEnvDeclarationIsOmitted(t *testing.T) {
	role, err := ParseRole("roles/x.md", "body", map[string]string{"fleet_id": "codellm"})
	require.NoError(t, err)

	bag := decodeBag(t, buildInputBag(role, renderCtx{}))

	require.Empty(t, bag.EnvPassthrough)
	require.Empty(t, bag.EnvPrefix)
}
