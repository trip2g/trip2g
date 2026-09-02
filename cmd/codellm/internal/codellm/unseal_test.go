package codellm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"trip2g/internal/dataencryption"
)

const testKey = "12345678901234567890123456789012" // 32 bytes

func sealWith(t *testing.T, plaintext string) string {
	t.Helper()
	blob, err := Seal(testKey, plaintext)
	require.NoError(t, err)
	return blob
}

func bagJSON(t *testing.T, v map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}

func TestSealRoundTrip(t *testing.T) {
	blob := sealWith(t, "s3cret")
	require.Greater(t, len(blob), len(sealedPrefix))
	require.Contains(t, blob, sealedPrefix)

	got, err := openSealed(testKey, blob)
	require.NoError(t, err)
	require.Equal(t, "s3cret", got)
}

// Each seal must use a fresh nonce, or identical secrets would be
// distinguishable in the vault by their ciphertext alone.
func TestSealIsNotDeterministic(t *testing.T) {
	first, second := sealWith(t, "same"), sealWith(t, "same")
	require.NotEqual(t, first, second)
}

func TestOpenSealedRejectsWrongKey(t *testing.T) {
	blob := sealWith(t, "s3cret")
	_, err := openSealed("00000000000000000000000000000000", blob)
	require.Error(t, err)
}

func TestSealRejectsShortKey(t *testing.T) {
	_, err := Seal("too-short", "x")
	require.Error(t, err)
}

func TestUnsealBag_OpensDeclaredFields(t *testing.T) {
	blob := sealWith(t, "krisp-token-value")
	bag := bagJSON(t, map[string]any{
		"frontmatter": map[string]string{
			"krisp_token":    blob,
			"krisp_base_url": "https://api.krisp.ai",
		},
		"unseal": []string{"krisp_token"},
		"depth":  1,
	})

	out, opened, err := unsealBag(bag, mapEnv{"SEAL_KEY": testKey}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"krisp-token-value"}, opened)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	secrets := got["secrets"].(map[string]any)
	require.Equal(t, "krisp-token-value", secrets["krisp_token"])

	fm := got["frontmatter"].(map[string]any)
	require.Equal(t, blob, fm["krisp_token"], "the ciphertext stays in the frontmatter")
	require.Equal(t, "https://api.krisp.ai", fm["krisp_base_url"])
	require.EqualValues(t, 1, got["depth"], "unrelated bag fields survive the round trip")
}

// A bag from a newer fleet may carry fields this codellm does not model. They
// must survive rather than be dropped by a typed round trip.
func TestUnsealBag_PreservesUnknownFields(t *testing.T) {
	bag := bagJSON(t, map[string]any{
		"frontmatter":   map[string]string{},
		"future_field":  "keep me",
		"changed_files": []any{},
	})

	out, _, err := unsealBag(bag, mapEnv{}, nil, nil)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.Equal(t, "keep me", got["future_field"])
}

func TestUnsealBag_NoDeclarationIsANoOp(t *testing.T) {
	bag := bagJSON(t, map[string]any{"frontmatter": map[string]string{"a": "b"}})

	out, opened, err := unsealBag(bag, mapEnv{}, nil, nil)
	require.NoError(t, err)
	require.Empty(t, opened)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	require.NotContains(t, got, "secrets")
}

func TestUnsealBag_CustomEnvKey(t *testing.T) {
	blob := sealWith(t, "v")
	bag := bagJSON(t, map[string]any{
		"frontmatter":    map[string]string{"tok": blob},
		"unseal":         []string{"tok"},
		"unseal_env_key": "SEAL_KEY_V2",
	})

	_, opened, err := unsealBag(bag, mapEnv{"SEAL_KEY_V2": testKey}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"v"}, opened)
}

// A declared field that is absent must fail as loudly as one that cannot be
// opened: both mean the role will not work, and silence would surface as a 401
// from the upstream API minutes later.
func TestUnsealBag_MissingFieldIsAnError(t *testing.T) {
	bag := bagJSON(t, map[string]any{
		"frontmatter": map[string]string{},
		"unseal":      []string{"absent"},
	})

	_, _, err := unsealBag(bag, mapEnv{"SEAL_KEY": testKey}, nil, nil)
	require.ErrorContains(t, err, "absent")
}

func TestUnsealBag_UnknownEnvKeyIsAnError(t *testing.T) {
	blob := sealWith(t, "v")
	bag := bagJSON(t, map[string]any{
		"frontmatter": map[string]string{"tok": blob},
		"unseal":      []string{"tok"},
	})

	_, _, err := unsealBag(bag, mapEnv{}, nil, nil)
	require.Error(t, err)
}

// The error must not echo what went wrong with the key material: a note author
// could otherwise probe codellm's environment for which vars exist and how long
// they are.
func TestUnsealBag_ErrorDoesNotLeakEnvDetail(t *testing.T) {
	blob := sealWith(t, "v")
	bag := bagJSON(t, map[string]any{
		"frontmatter": map[string]string{"tok": blob},
		"unseal":      []string{"tok"},
	})

	_, _, err := unsealBag(bag, mapEnv{"SEAL_KEY": "short"}, nil, nil)

	require.Error(t, err)
	require.NotContains(t, err.Error(), "32 bytes")
	require.NotContains(t, err.Error(), "short")
	require.NotContains(t, err.Error(), "SEAL_KEY")
}

// The key must never be handed to the code child: that would let any role open
// any blob it can read, which is the whole point of unsealing in Go.
func TestUnsealBag_RefusesKeyExposedToChildren(t *testing.T) {
	blob := sealWith(t, "v")
	bag := bagJSON(t, map[string]any{
		"frontmatter": map[string]string{"tok": blob},
		"unseal":      []string{"tok"},
	})

	_, _, err := unsealBag(bag, mapEnv{"SEAL_KEY": testKey}, []string{"SEAL_KEY"}, nil)

	require.ErrorIs(t, err, errSealKeyExposed)
}

// An operator prefix forwards every matching var into the child env, so a key
// it covers is just as exposed as one listed by name. Startup only checks the
// default name; a role-named key is only known here.
func TestUnsealBag_RefusesKeyCoveredByExposedPrefix(t *testing.T) {
	blob := sealWith(t, "v")

	tests := []struct {
		name     string
		envKey   string
		prefixes []string
		wantErr  error
	}{
		{name: "role key under operator prefix", envKey: "KRISP_SEAL_KEY", prefixes: []string{"KRISP_"}, wantErr: errSealKeyExposed},
		{name: "default key under operator prefix", envKey: "", prefixes: []string{"SEAL"}, wantErr: errSealKeyExposed},
		{name: "prefix does not cover the key", envKey: "KRISP_SEAL_KEY", prefixes: []string{"OTHER_"}},
		{name: "empty prefix covers nothing", envKey: "KRISP_SEAL_KEY", prefixes: []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := map[string]any{
				"frontmatter": map[string]string{"tok": blob},
				"unseal":      []string{"tok"},
			}
			if tt.envKey != "" {
				doc["unseal_env_key"] = tt.envKey
			}
			env := mapEnv{"SEAL_KEY": testKey, "KRISP_SEAL_KEY": testKey}

			_, opened, err := unsealBag(bagJSON(t, doc), env, nil, tt.prefixes)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"v"}, opened)
		})
	}
}

func TestRedactSecrets(t *testing.T) {
	in := `parse stdout: (got: "{\"tok\": \"s3cret\"}")`
	require.NotContains(t, redactSecrets(in, []string{"s3cret"}), "s3cret")
	require.Contains(t, redactSecrets(in, []string{"s3cret"}), redacted)
	require.Equal(t, in, redactSecrets(in, nil))
	require.Equal(t, in, redactSecrets(in, []string{""}), "an empty secret must not redact everything")
}

func TestDataEncryptionCompat(t *testing.T) {
	// Seal must produce something the shared manager can open: the format is
	// nonce||ciphertext from internal/dataencryption, only base64'd and tagged.
	m, err := dataencryption.NewManager(dataencryption.Config{Key: testKey})
	require.NoError(t, err)

	raw, err := decodeSealed(sealWith(t, "hello"))
	require.NoError(t, err)
	plain, err := m.DecryptData(raw)
	require.NoError(t, err)
	require.Equal(t, "hello", string(plain))
}

// A bag that is not a JSON object declares nothing to unseal and must travel
// untouched — codellm accepted arbitrary bag bytes before sealing existed.
func TestUnsealBag_NonJSONPassesThrough(t *testing.T) {
	raw := []byte("hello, not json")

	out, opened, err := unsealBag(raw, mapEnv{}, nil, nil)

	require.NoError(t, err)
	require.Empty(t, opened)
	require.Equal(t, raw, out)
}

func TestValidateSealKeyNotExposed(t *testing.T) {
	require.NoError(t, ValidateSealKeyNotExposed(nil, nil))
	require.NoError(t, ValidateSealKeyNotExposed([]string{"KRISP_TOKEN"}, []string{"KRISP_"}))

	require.ErrorIs(t, ValidateSealKeyNotExposed([]string{"SEAL_KEY"}, nil), errSealKeyExposed)
	require.ErrorIs(t, ValidateSealKeyNotExposed(nil, []string{"SEAL"}), errSealKeyExposed,
		"a prefix sweeps the key in just as effectively as naming it")
	require.ErrorIs(t, ValidateSealKeyNotExposed(nil, []string{"S"}), errSealKeyExposed)
}
