package graph

// The SPA user page ships GraphQL operations for app authors to copy. Every
// ```graphql block must validate against the real schema and stay under the
// server's complexity limit, and every ```json block after it must be valid
// variables for that operation. The en and ru pages carry identical blocks.

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"testing"

	"trip2g/internal/graph/generated"

	"github.com/99designs/gqlgen/complexity"
	"github.com/stretchr/testify/require"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/validator"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

type docOperation struct {
	query     string
	variables []string
}

func readDocOperations(t *testing.T, page string) []docOperation {
	t.Helper()

	data, err := os.ReadFile("../../docs/" + page + ".md")
	require.NoError(t, err)

	fence := regexp.MustCompile("(?ms)^```(graphql|json)\n(.*?)^```$")

	var ops []docOperation
	for _, m := range fence.FindAllStringSubmatch(string(data), -1) {
		if m[1] == "graphql" {
			ops = append(ops, docOperation{query: m[2]})
			continue
		}
		require.NotEmpty(t, ops, "docs/%s.md: a json block before any graphql block", page)
		last := &ops[len(ops)-1]
		last.variables = append(last.variables, m[2])
	}

	return ops
}

func TestSPADocOperationsMatchSchema(t *testing.T) {
	es := generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{}})

	en := readDocOperations(t, "en/user/spa")
	ru := readDocOperations(t, "ru/user/spa")
	require.NotEmpty(t, en)
	require.Equal(t, en, ru, "en and ru spa pages must carry identical graphql and json blocks")

	for _, op := range en {
		doc, errs := gqlparser.LoadQueryWithRules(es.Schema(), op.query, rules.NewDefaultRules())
		require.Empty(t, errs, op.query)
		require.Len(t, doc.Operations, 1, op.query)

		operation := doc.Operations[0]

		allVars := []map[string]any{{}}
		for _, raw := range op.variables {
			var vars map[string]any
			require.NoError(t, json.Unmarshal([]byte(raw), &vars), raw)

			coerced, varErr := validator.VariableValues(es.Schema(), operation, vars)
			require.NoError(t, varErr, raw)

			allVars = append(allVars, coerced)
		}

		for _, vars := range allVars {
			cost := complexity.Calculate(context.Background(), es, operation, vars)
			require.LessOrEqual(t, cost, maxQueryComplexity, op.query)
		}
	}
}

func readDocInlineOperations(t *testing.T, page string) []string {
	t.Helper()

	data, err := os.ReadFile("../../docs/" + page + ".md")
	require.NoError(t, err)

	inline := regexp.MustCompile("(?s)(?:make(?:Request|Subscription)|askTrip2g)\\(`(.*?)`\\)")

	var ops []string
	for _, m := range inline.FindAllStringSubmatch(string(data), -1) {
		ops = append(ops, m[1])
	}

	return ops
}

func TestSPADocInlineOperationsMatchSchema(t *testing.T) {
	es := generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{}})

	en := readDocInlineOperations(t, "en/user/spa")
	ru := readDocInlineOperations(t, "ru/user/spa")
	require.Len(t, en, 5)
	require.Equal(t, en, ru, "en and ru spa pages must carry identical inline operations")

	for _, op := range en {
		doc, errs := gqlparser.LoadQueryWithRules(es.Schema(), op, rules.NewDefaultRules())
		require.Empty(t, errs, op)
		require.Len(t, doc.Operations, 1, op)

		cost := complexity.Calculate(context.Background(), es, doc.Operations[0], map[string]any{})
		require.LessOrEqual(t, cost, maxQueryComplexity, op)
	}
}
