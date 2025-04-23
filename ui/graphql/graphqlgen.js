module.exports = {
   schema: 'http://localhost:8081/graphql',
   documents: [__dirname + '/../**/*.ts'],
   generates: {
      [__dirname + '/queries.ts']: {
        plugins: [__dirname + '/graphqlmol.js'],
      },
   },
}
