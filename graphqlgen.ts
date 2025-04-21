import type { CodegenConfig } from '@graphql-codegen/cli'

const config: CodegenConfig = {
   schema: 'http://localhost:8081/graphql',
   documents: ['ui/**/*.ts'],
   generates: {
      './ui/graphql/queries.ts': {
        plugins: [
          {
            add: {
              content: 'namespace $.$$ {',
            },
          },
          'typescript',
          'typescript-operations',
          './graphqlmol.js',
        ],
        config: {
          noExport: true,
          documentMode: 'string',
        },
      },
      // './ui/graphql/generated/': {
      //   preset: 'client',
      //   config: {
      //     documentMode: 'string',
      //   },
      // },
   },
}

export default config
