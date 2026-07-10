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

func TestShouldSkipTx_FragmentExcludedByInclude(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				...CronOps @include(if: false)
			}
		}
		fragment CronOps on AdminMutation {
			runCronJob(input: {id: 1}) {
				__typename
			}
		}
	`)

	require.False(t, shouldSkipTx(op, realSkipTxMap()),
		"@skipTx field in a fragment excluded by @include(if: false) never executes, tx must stay on")
}

func TestShouldSkipTx_FragmentExcludedBySkip(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				...CronOps @skip(if: true)
			}
		}
		fragment CronOps on AdminMutation {
			runCronJob(input: {id: 1}) {
				__typename
			}
		}
	`)

	require.False(t, shouldSkipTx(op, realSkipTxMap()),
		"@skipTx field in a fragment excluded by @skip(if: true) never executes, tx must stay on")
}

func TestShouldSkipTx_FieldExcludedBySkip(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				runCronJob(input: {id: 1}) @skip(if: true) {
					__typename
				}
			}
		}
	`)

	require.False(t, shouldSkipTx(op, realSkipTxMap()),
		"@skipTx field excluded by @skip(if: true) never executes, tx must stay on")
}

func TestShouldSkipTx_FragmentIncludedByLiteralTrue(t *testing.T) {
	op := loadTestOperation(t, `
		mutation {
			admin {
				...CronOps @include(if: true)
			}
		}
		fragment CronOps on AdminMutation {
			runCronJob(input: {id: 1}) {
				__typename
			}
		}
	`)

	require.True(t, shouldSkipTx(op, realSkipTxMap()),
		"@include(if: true) does not exclude the fragment, @skipTx must still be detected")
}

func TestShouldSkipTx_FragmentGuardedByVariable(t *testing.T) {
	op := loadTestOperation(t, `
		mutation ($run: Boolean!) {
			admin {
				...CronOps @include(if: $run)
			}
		}
		fragment CronOps on AdminMutation {
			runCronJob(input: {id: 1}) {
				__typename
			}
		}
	`)

	require.True(t, shouldSkipTx(op, realSkipTxMap()),
		"variable-guarded fragment may execute, must be treated as skipTx conservatively")
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
