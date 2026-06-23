package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"trip2g/internal/logger"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

const fullIntrospectionQuery = `{
  __schema {
    queryType { name }
    mutationType { name }
    types {
      kind name description
      fields(includeDeprecated: true) {
        name description
        args { name description type { kind name ofType { kind name ofType { kind name } } } defaultValue }
        type { kind name ofType { kind name ofType { kind name } } }
      }
      inputFields {
        name description type { kind name ofType { kind name ofType { kind name } } } defaultValue
      }
      interfaces { kind name ofType { kind name } }
      enumValues(includeDeprecated: true) { name description }
      possibleTypes { kind name }
    }
  }
}`

func handleGraphQLIntrospection(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[GraphQLIntrospectionArguments](argsRaw, id, "graphql_introspection")
	if errResp != nil {
		return *errResp
	}
	if args.Pattern == "" {
		return errorResponse(id, ErrCodeInvalidParams, "pattern is required")
	}

	raw, err := env.GraphQLRequest(ctx, fullIntrospectionQuery, nil)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Introspection failed: "+err.Error())
	}

	filtered, err := filterIntrospection(raw, args.Pattern)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "Filter failed: "+err.Error())
	}

	return successResponse(id, textToolResult(string(filtered)))
}

// graphqlReadOnlyRootFields is the allowlist of Query root fields accessible via graphql_request
// on the admin (non-federated) path.
var graphqlReadOnlyRootFields = map[string]bool{ //nolint:gochecknoglobals // immutable allowlist
	"note":             true,
	"search":           true,
	"similarNotes":     true,
	"viewer":           true,
	"notePaths":        true,
	"resolveWikilinks": true,
}

// graphqlFederatedRootFields is the subset of read-only root fields safe to
// expose to a federation peer: every field here enforces per-note access via
// canreadnote. notePaths/resolveWikilinks are deliberately excluded — they
// return unfiltered paths and would leak private structure outside the
// caller's granted subgraphs.
var graphqlFederatedRootFields = map[string]bool{ //nolint:gochecknoglobals // immutable allowlist
	"note":         true,
	"search":       true,
	"similarNotes": true,
	"viewer":       true,
}

// validateReadOnlyQuery parses the query and enforces:
//  1. Parse must succeed.
//  2. Every operation must be a query (not mutation or subscription).
//  3. Every top-level root field must be in the provided allowlist.
//     Inline fragments and fragment spreads at root are rejected.
//
// The function is intentionally standalone and reusable by other tools
// (e.g. federated_graphql_request). Pass graphqlReadOnlyRootFields for the
// admin path and graphqlFederatedRootFields for the federated/scoped path.
func validateReadOnlyQuery(query string, allowed map[string]bool) error {
	doc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		return fmt.Errorf("query parse error: %w", err)
	}
	for _, op := range doc.Operations {
		if op.Operation != ast.Query {
			return fmt.Errorf("only queries are allowed, got %s", op.Operation)
		}
		for _, sel := range op.SelectionSet {
			field, ok := sel.(*ast.Field)
			if !ok {
				return errors.New("only field selections are allowed at query root")
			}
			if !allowed[field.Name] {
				return fmt.Errorf("root field %q is not allowed", field.Name)
			}
		}
	}
	return nil
}

func handleGraphQLRequestScoped(ctx context.Context, env Env, id any, argsRaw json.RawMessage, allowed []string) Response {
	args, errResp := unmarshalArgs[GraphQLRequestArguments](argsRaw, id, "graphql_request")
	if errResp != nil {
		return *errResp
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	if err := validateReadOnlyQuery(args.Query, graphqlFederatedRootFields); err != nil {
		return errorResponse(id, ErrCodeInvalidParams, "query rejected: "+err.Error())
	}
	log := logger.WithPrefix(env.Logger(), "mcp:handleGraphQLRequestScoped")
	log.Debug("graphql_request scoped accepted", "query", args.Query)

	result, err := env.GraphQLRequestScoped(ctx, args.Query, args.Variables, allowed)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "GraphQL request failed: "+err.Error())
	}

	var parsed any
	if jsonErr := json.Unmarshal(result, &parsed); jsonErr != nil {
		return successResponse(id, textToolResult(string(result)))
	}
	return successResponse(id, structuredToolResult("structured result", parsed))
}

func handleGraphQLRequest(ctx context.Context, env Env, id any, argsRaw json.RawMessage) Response {
	args, errResp := unmarshalArgs[GraphQLRequestArguments](argsRaw, id, "graphql_request")
	if errResp != nil {
		return *errResp
	}
	if args.Query == "" {
		return errorResponse(id, ErrCodeInvalidParams, "query is required")
	}

	if err := validateReadOnlyQuery(args.Query, graphqlReadOnlyRootFields); err != nil {
		return errorResponse(id, ErrCodeInvalidParams, "query rejected: "+err.Error())
	}
	log := logger.WithPrefix(env.Logger(), "mcp:handleGraphQLRequest")
	log.Debug("graphql_request accepted", "query", args.Query)

	result, err := env.GraphQLRequest(ctx, args.Query, args.Variables)
	if err != nil {
		return errorResponse(id, ErrCodeInternal, "GraphQL request failed: "+err.Error())
	}

	var parsed any
	if jsonErr := json.Unmarshal(result, &parsed); jsonErr != nil {
		return successResponse(id, textToolResult(string(result)))
	}
	return successResponse(id, structuredToolResult("structured result", parsed))
}

type introspectionTypeRef struct {
	Kind   string                `json:"kind"`
	Name   string                `json:"name"`
	OfType *introspectionTypeRef `json:"ofType,omitempty"`
}

type introspectionField struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Args        []introspectionArg   `json:"args,omitempty"`
	Type        introspectionTypeRef `json:"type"`
}

type introspectionArg struct {
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Type         introspectionTypeRef `json:"type"`
	DefaultValue *string              `json:"defaultValue,omitempty"`
}

type introspectionEnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type introspectionType struct {
	Kind          string                   `json:"kind"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	Fields        []introspectionField     `json:"fields,omitempty"`
	InputFields   []introspectionArg       `json:"inputFields,omitempty"`
	Interfaces    []introspectionTypeRef   `json:"interfaces,omitempty"`
	EnumValues    []introspectionEnumValue `json:"enumValues,omitempty"`
	PossibleTypes []introspectionTypeRef   `json:"possibleTypes,omitempty"`
}

func typeRefName(ref *introspectionTypeRef) string {
	if ref == nil {
		return ""
	}
	if ref.Name != "" {
		return ref.Name
	}
	return typeRefName(ref.OfType)
}

func collectReferencedNames(t introspectionType, out map[string]bool) {
	for _, f := range t.Fields {
		out[typeRefName(&f.Type)] = true
		for _, a := range f.Args {
			out[typeRefName(&a.Type)] = true
		}
	}
	for _, f := range t.InputFields {
		out[typeRefName(&f.Type)] = true
	}
	for _, i := range t.Interfaces {
		ref := i
		out[typeRefName(&ref)] = true
	}
	for _, p := range t.PossibleTypes {
		ref := p
		out[typeRefName(&ref)] = true
	}
}

func compilePattern(pattern string) *regexp.Regexp {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return regexp.MustCompile("(?i)" + regexp.QuoteMeta(pattern))
	}
	return re
}

func seedMatchingTypes(allTypes []introspectionType, re *regexp.Regexp) map[string]bool {
	needed := map[string]bool{}
	for _, t := range allTypes {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}
		if re.MatchString(t.Name) {
			needed[t.Name] = true
			continue
		}
		for _, f := range t.Fields {
			if re.MatchString(f.Name) {
				needed[t.Name] = true
				break
			}
		}
	}
	return needed
}

func expandReferenced(needed map[string]bool, typeByName map[string]introspectionType) {
	for changed := true; changed; {
		changed = false
		for name := range needed {
			t, ok := typeByName[name]
			if !ok {
				continue
			}
			refs := map[string]bool{}
			collectReferencedNames(t, refs)
			for ref := range refs {
				if ref == "" || strings.HasPrefix(ref, "__") || needed[ref] {
					continue
				}
				needed[ref] = true
				changed = true
			}
		}
	}
}

func filterIntrospection(data []byte, pattern string) ([]byte, error) {
	var wrapper struct {
		Data struct {
			Schema struct {
				QueryType *struct {
					Name string `json:"name"`
				} `json:"queryType"`
				MutationType *struct {
					Name string `json:"name"`
				} `json:"mutationType"`
				Types []introspectionType `json:"types"`
			} `json:"__schema"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	re := compilePattern(pattern)
	allTypes := wrapper.Data.Schema.Types
	typeByName := make(map[string]introspectionType, len(allTypes))
	for _, t := range allTypes {
		typeByName[t.Name] = t
	}

	needed := seedMatchingTypes(allTypes, re)
	expandReferenced(needed, typeByName)

	filtered := make([]introspectionType, 0, len(needed))
	for _, t := range allTypes {
		if needed[t.Name] {
			filtered = append(filtered, t)
		}
	}
	wrapper.Data.Schema.Types = filtered

	return json.Marshal(wrapper)
}
