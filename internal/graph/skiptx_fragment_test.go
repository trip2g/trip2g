package graph

// @skipTx fields must be detected regardless of selection shape: a mutation
// reaching runCronJob through a fragment spread or inline fragment must NOT
// open the request transaction, or it deadlocks against the single writer.

import (
	"testing"

	"github.com/stretchr/testify/require"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

// loadTestOperation parses and validates query against the real executable
// schema, so FragmentSpread.Definition is resolved exactly as gqlgen does it.
func loadTestOperation(t *testing.T, query string) *ast.OperationDefinition {
	t.Helper()

	schema := NewExecutableSchema(Config{Resolvers: &Resolver{}})
	doc, errList := gqlparser.LoadQueryWithRules(schema.Schema(), query, rules.NewDefaultRules())
	require.Empty(t, errList)
	require.Len(t, doc.Operations, 1)

	return doc.Operations[0]
}

func realSkipTxMap() map[string]struct{} {
	return buildSkipTxMap(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
}

func TestShouldSkipTx_FragmentSpread(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				...CronOps
			}
		}
		fragment CronOps on AdminMutation {
			runCronJob(input: {id: 1}) {
				__typename
			}
		}
	`)

	require.True(t, shouldSkipTx(op, realSkipTxMap()),
		"@skipTx field selected via fragment spread must be detected")
}

func TestShouldSkipTx_InlineFragment(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				... on AdminMutation {
					runCronJob(input: {id: 1}) {
						__typename
					}
				}
			}
		}
	`)

	require.True(t, shouldSkipTx(op, realSkipTxMap()),
		"@skipTx field selected via inline fragment must be detected")
}

func TestShouldSkipTx_FragmentWithoutSkipTxField(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			...RootOps
		}
		fragment RootOps on Mutation {
			signOut {
				__typename
			}
		}
	`)

	require.False(t, shouldSkipTx(op, realSkipTxMap()),
		"fragment without @skipTx fields must still open the transaction")
}
