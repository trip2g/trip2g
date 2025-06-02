module.exports = {
	schema: 'http://localhost:8081/graphql',
  documents: ['assets/ui/**/*.ts'],
	pluckConfig: {
		globalGqlIdentifierName: ['gql', '$trip2g_graphql_request', '$trip2g_graphql_subscription'],
	},
	generates: {
    ['assets/ui/graphql/queries.ts']: {
			plugins: [
				{
					add: {
						content: 'namespace $.$$ {\n\n',
					},
				},
				'typescript',
				'typescript-operations',
        './graphqlmol.js',
			],
			config: {
				// noExport: true,
				molPrefix: '$trip2g_graphql',
			},
		},
	},
}
