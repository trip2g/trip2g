// GraphQL codegen for the frontend: .graphql files co-located with $mol
// components → per-file <name>.graphql.ts typed wrappers in `namespace $`.
// Ported from the graphqlgen.js of github.com/trip2g/mol_graphql.
//
// Schema comes from the checked-in SDL files — NO live introspection, no dev
// server needed (unlike the retired legacy root graphqlgen.js, which needed a
// running server).
//
// Endpoint routing is by file suffix, not directory:
//   *.graphql          → main endpoint  (/_system/graphql),        runtime $trip2g_gql
//   *.codellm.graphql  → codellm subgraph (/_system/codellm/graphql), runtime $trip2g_codellm_gql
//
// The main block owns the shared schema types (enums, inputs, Scalars,
// Maybe/Exact helpers) in assets/ui/gql/schema.graphql.ts, declared once in
// `namespace $` and referenced by every generated file.
//
// The codellm block has a different schema, so it emits its own schema types
// with `onlyOperationTypes` (enums + inputs only) into
// assets/ui/codellm/gql/schema.graphql.ts, strips the helper aliases the main
// block already declares (molStripHelpers), and renames `Scalars`
// (molScalarsName) to avoid the duplicate-identifier collision in `namespace $`.

const base = {
	// repo-relative assets/ui/user/favoritenote/notes.graphql is workspace
	// trip2g/user/favoritenote/notes = $trip2g_user_favoritenote_notes
	molPackage: 'trip2g',
	molRoot: 'assets/ui',
	// operation names are auto-derived from the file path; type names keep
	// GraphQL casing as-is (matches molplugin's resultType computation)
	namingConvention: 'keep',
	// Relay-style fragment masking: a spread field is typed as an opaque ref
	inlineFragmentTypes: 'mask',
	// every mutation refetches every query — trip2g's existing reset_marker
	// convention, shared with the legacy $trip2g_graphql_raw_request path
	revalidation: 'all',
}

module.exports = {
	generates: {
		'./': {
			schema: 'internal/graph/schema.graphqls',
			preset: require('./preset.js'),
			plugins: [], // per-file plugin chains are built by the preset
			documents: ['assets/ui/**/*.graphql', '!assets/ui/**/*.codellm.graphql'],
			config: {
				...base,
				molRuntime: '$trip2g_gql',
				molSchemaTypes: 'assets/ui/gql/schema.graphql.ts',
			},
		},
		'./codellm-subgraph': {
			schema: 'cmd/codellm/internal/codellm/codellmgql/schema.graphqls',
			preset: require('./preset.js'),
			plugins: [],
			documents: ['assets/ui/**/*.codellm.graphql'],
			config: {
				...base,
				molRuntime: '$trip2g_codellm_gql',
				molSchemaTypes: 'assets/ui/codellm/gql/schema.graphql.ts',
				onlyOperationTypes: true, // enums + inputs, no object types (inlined in results anyway)
				molStripHelpers: true, // Maybe/Exact/... already declared by the main schema types
				molScalarsName: '$trip2g_codellm_gql_scalars', // avoid duplicate `Scalars`
			},
		},
	},
}
