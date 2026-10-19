package graph

import (
	"context"
	"strings"
	"testing"

	"trip2g/internal/graph/generated"

	"github.com/99designs/gqlgen/complexity"
	"github.com/stretchr/testify/require"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

func TestNoteQueryComplexity(t *testing.T) {
	es := costedSchema{generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{}})}

	cost := func(calls int) int {
		var q strings.Builder
		q.WriteString("{")
		for i := range calls {
			q.WriteString(" q" + string(rune('a'+i)) + `: noteQuery(input: {query: "q"}) { ... on NoteQueryPayload { result } }`)
		}
		q.WriteString(" }")

		doc, errs := gqlparser.LoadQueryWithRules(es.Schema(), q.String(), rules.NewDefaultRules())
		require.Empty(t, errs)
		return complexity.Calculate(context.Background(), es, doc.Operations[0], map[string]any{})
	}

	require.LessOrEqual(t, cost(2), maxQueryComplexity)
	require.Greater(t, cost(3), maxQueryComplexity)
}
