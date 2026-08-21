package codellm

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"trip2g/internal/dataencryption"
)

// sealedPrefix tags a frontmatter value as ciphertext and carries the format
// version, so the algorithm can change later without invalidating blobs that
// already sit in vaults.
const sealedPrefix = "sealed:v1:"

// DefaultSealEnvKey is the env var a role's unseal_env_key defaults to. Exported
// so the seal CLI defaults to the same name the service resolves.
const DefaultSealEnvKey = "SEAL_KEY"

const defaultSealEnvKey = DefaultSealEnvKey

// redacted replaces a secret wherever one would otherwise reach an operator.
const redacted = "[redacted]"

// errSealKeyExposed is the refusal to unseal while the key is also being handed
// to the code child. That combination silently undoes the whole design: any
// role could then open any blob it can read, which is exactly what unsealing in
// Go rather than in the sandbox exists to prevent.
var errSealKeyExposed = errors.New("seal key is exposed to code; refusing to unseal")

// errUnsealFailed is the single outward-facing failure. It is deliberately
// uninformative: the note names the env var, so a detailed error would let a
// note author probe codellm's environment for which vars exist and how long
// they are. The detail goes to codellm's own log instead.
var errUnsealFailed = errors.New("unseal failed")

// envLookup is the environment the key is resolved against: codellm's own
// process env in production, a map in tests.
type envLookup interface {
	get(name string) string
}

// mapEnv is the test-side envLookup.
type mapEnv map[string]string

func (e mapEnv) get(name string) string { return e[name] }

// Seal produces the value pasted into a role note's frontmatter. Exported
// because the seal CLI and the seal endpoint are the only two ways to make one.
func Seal(key, plaintext string) (string, error) {
	m, err := dataencryption.NewManager(dataencryption.Config{Key: key})
	if err != nil {
		return "", err
	}
	ciphertext, err := m.EncryptData([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return sealedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decodeSealed(blob string) ([]byte, error) {
	body, ok := strings.CutPrefix(blob, sealedPrefix)
	if !ok {
		return nil, fmt.Errorf("value is not tagged %s", sealedPrefix)
	}
	return base64.StdEncoding.DecodeString(body)
}

func openSealed(key, blob string) (string, error) {
	raw, err := decodeSealed(blob)
	if err != nil {
		return "", err
	}
	m, err := dataencryption.NewManager(dataencryption.Config{Key: key})
	if err != nil {
		return "", err
	}
	plain, err := m.DecryptData(raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// unsealBag opens the frontmatter fields the role declared and returns the bag
// with a "secrets" section added, plus the plaintexts it opened so the caller
// can keep them out of anything it reports.
//
// The bag is handled as a generic map rather than fleetinput.Input on purpose:
// fleet and codellm are deployed separately, so a newer fleet may send fields
// this build does not model, and a typed round trip would silently drop them.
//
// exposedToChild is the env allowlist the code child will receive; naming the
// seal key there is a deploy mistake that must fail, not be worked around.
func unsealBag(bag []byte, env envLookup, exposedToChild []string) ([]byte, []string, error) {
	if len(bag) == 0 {
		return bag, nil, nil
	}
	// The bag is opaque to codellm unless it is a JSON object: a caller may send
	// anything, and a bag that is not one declares no fields to unseal. Failing
	// here would break requests that worked before sealing existed.
	var parsed any
	_ = json.Unmarshal(bag, &parsed)
	doc, isObject := parsed.(map[string]any)
	if !isObject {
		return bag, nil, nil
	}

	fields := stringSlice(doc["unseal"])
	if len(fields) == 0 {
		return bag, nil, nil
	}

	envKey := defaultSealEnvKey
	if v, ok := doc["unseal_env_key"].(string); ok && strings.TrimSpace(v) != "" {
		envKey = strings.TrimSpace(v)
	}
	if nameIsExposed(envKey, exposedToChild) {
		return nil, nil, fmt.Errorf("%w: %s", errSealKeyExposed, envKey)
	}

	frontmatter, _ := doc["frontmatter"].(map[string]any)
	key := env.get(envKey)
	secrets := make(map[string]string, len(fields))
	opened := make([]string, 0, len(fields))

	for _, field := range fields {
		blob, isString := frontmatter[field].(string)
		if !isString {
			// As loud as an unopenable value: both mean the role cannot work.
			return nil, nil, fmt.Errorf("%w: %q is not in the frontmatter", errUnsealFailed, field)
		}
		plain, err := openSealed(key, blob)
		if err != nil {
			return nil, nil, fmt.Errorf("%w for %q", errUnsealFailed, field)
		}
		secrets[field] = plain
		opened = append(opened, plain)
	}

	doc["secrets"] = secrets
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, nil, fmt.Errorf("delivery bag: %w", err)
	}
	return out, opened, nil
}

// nameIsExposed reports whether name is on the child env allowlist. Only exact
// names are checked here; prefixes are rejected at startup, where the operator
// can act on them.
func nameIsExposed(name string, exposed []string) bool {
	for _, e := range exposed {
		if e == name {
			return true
		}
	}
	return false
}

func stringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, isString := item.(string); isString && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// redactSecrets removes opened plaintexts from text bound for an operator.
//
// Unlike scanning for "secrets" in general — which cannot work against a role
// that wants to leak one — this is exact: codellm knows the strings it just
// decrypted. It matters because ParseCodeOutput quotes up to 200 bytes of the
// child's stdout when the JSON contract fails to parse, and the likeliest way
// to break that contract is printing the bag while debugging. Without this the
// token rides that preview into the 422, into fleet's run log, and into
// trip2g's delivery-log store — the store this whole feature exists to keep the
// secret out of.
func redactSecrets(text string, secrets []string) string {
	for _, s := range secrets {
		if s == "" {
			continue
		}
		text = strings.ReplaceAll(text, s, redacted)
	}
	return text
}

// osEnv resolves the seal key against codellm's own process environment.
type osEnv struct{}

func (osEnv) get(name string) string { return os.Getenv(name) }

// ValidateSealKeyNotExposed refuses a configuration that forwards the default
// seal key to executed code.
//
// It is checked at startup because it is a deploy mistake, and a silent one:
// listing SEAL_KEY in the expose allowlist — or letting a prefix sweep it in —
// leaves every role able to open every blob, with nothing visibly broken. Only
// the default name can be checked here; a role naming its own key is checked
// per run, since codellm learns those names only from notes.
func ValidateSealKeyNotExposed(exposeEnv, exposeEnvPrefix []string) error {
	if nameIsExposed(defaultSealEnvKey, exposeEnv) {
		return fmt.Errorf("%w: %s is in the expose-env allowlist", errSealKeyExposed, defaultSealEnvKey)
	}
	for _, p := range exposeEnvPrefix {
		if p != "" && strings.HasPrefix(defaultSealEnvKey, p) {
			return fmt.Errorf("%w: expose-env-prefix %q covers %s", errSealKeyExposed, p, defaultSealEnvKey)
		}
	}
	return nil
}

// OpenSealedForTest opens a blob. Exported for the seal CLI's test, which has to
// prove that what the command prints is what the running service can read.
func OpenSealedForTest(key, blob string) (string, error) { return openSealed(key, blob) }
