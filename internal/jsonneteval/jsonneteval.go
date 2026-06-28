// Package jsonneteval is the single source of truth for evaluating jsonnet
// snippets against JSON ext-vars (outbound webhook transforms, frontmatter
// patches). It owns the safe VM stack limit so every caller is consistent.
package jsonneteval

import (
	"encoding/json"
	"fmt"

	jsonnet "github.com/google/go-jsonnet"
)

// NewVM returns a jsonnet VM with a safe stack limit (no IO; go-jsonnet is pure).
func NewVM() *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500 // Prevent stack overflow from recursive jsonnet.
	return vm
}

// EvalJSON evaluates src with the given ext-vars bound via std.extVar and
// returns the result as raw JSON. Each ext-var value is bound verbatim (the
// caller decides whether a value is a JSON string to std.parseJson).
func EvalJSON(src string, extVars map[string]string) (json.RawMessage, error) {
	vm := NewVM()
	for k, v := range extVars {
		vm.ExtVar(k, v)
	}

	out, err := vm.EvaluateAnonymousSnippet("transform", src)
	if err != nil {
		return nil, fmt.Errorf("evaluate jsonnet: %w", err)
	}

	if !json.Valid([]byte(out)) {
		return nil, fmt.Errorf("jsonnet output is not valid JSON")
	}

	return json.RawMessage(out), nil
}

// Validate compiles and runs src against sampleExtVars, discarding the result.
// Used at CRUD time to reject a transform that cannot even evaluate.
func Validate(src string, sampleExtVars map[string]string) error {
	_, err := EvalJSON(src, sampleExtVars)
	return err
}
