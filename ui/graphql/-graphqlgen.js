module.exports = {
	schema: 'http://localhost:8081/graphql',
	documents: [__dirname + '/../**/*.ts'],
	generates: {
		[__dirname + '/queries.ts']: {
			plugins: [
				{
					add: {
						content: 'namespace $.$$ {\n\n',
					},
				},
				'typescript',
				'typescript-operations',
				__dirname + '/-graphqlmol.js',
			],
			config: {
				noExport: true,
			},
		},
	},
}
