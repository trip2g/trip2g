module.exports = {
   schema: 'http://localhost:8081/graphql',
   documents: [__dirname + '/../**/*.ts'],
   generates: {
      [__dirname + '/-/']: {
        preset: 'client',
        // plugins: [__dirname + '/-graphqlmol.js'],
      },
   },
}
