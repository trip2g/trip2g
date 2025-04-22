import type { CodegenConfig } from '@graphql-codegen/cli'

const config: CodegenConfig = {
   schema: 'http://localhost:8081/graphql',
   documents: ['ui/**/*.ts'],
   generates: {
      './ui/graphql/queries.ts': {
        plugins: [
          './graphqlmol.js',
        ],
      },
   },
}

export default config
