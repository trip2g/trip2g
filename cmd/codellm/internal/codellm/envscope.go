package codellm

import (
	"encoding/json"
	"strings"
)

// effectiveEnvNames resolves which of codellm's own env vars reach the code
// child: the operator's allowlist intersected with what the role declared.
//
// The operator's list is the boundary and a role can only narrow it. A role
// that declares nothing keeps the behaviour that existed before the field came
// back — it sees the whole allowlist — so restoring the field breaks no
// existing role note.
//
// Both sides accept names and prefixes, and both are resolved against the real
// variable names rather than against each other. That is what makes an operator
// prefix and a role prefix of different lengths compose correctly, and it means
// the result handed downstream is an exact list with no prefix left to expand.
func effectiveEnvNames(environ, operatorNames, operatorPrefixes, roleNames, rolePrefixes []string) []string {
	roleDeclaredNothing := len(roleNames) == 0 && len(rolePrefixes) == 0

	var out []string
	for _, kv := range environ {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if !matchesAllowlist(name, operatorNames, operatorPrefixes) {
			continue
		}
		if roleDeclaredNothing || matchesAllowlist(name, roleNames, rolePrefixes) {
			out = append(out, name)
		}
	}
	return out
}

func matchesAllowlist(name string, names, prefixes []string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// envDeclaration reads env_passthrough / env_prefix out of the delivery bag.
// A bag that is not a JSON object, or carries neither field, declares nothing.
func envDeclaration(bag []byte) ([]string, []string) {
	var parsed any
	_ = json.Unmarshal(bag, &parsed)
	doc, isObject := parsed.(map[string]any)
	if !isObject {
		return nil, nil
	}
	return stringSlice(doc["env_passthrough"]), stringSlice(doc["env_prefix"])
}
