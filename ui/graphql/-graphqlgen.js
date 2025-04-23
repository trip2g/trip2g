module.exports = {
   schema: 'http://localhost:8081/graphql',
   documents: [__dirname + '/../**/*.ts'],
   generates: {
      [__dirname + '/queries.ts']: {
        plugins: [
          // add
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
        // plugins: [__dirname + '/-graphqlmol.js'],
      },
   },
}
