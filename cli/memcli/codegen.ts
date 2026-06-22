import type { CodegenConfig } from '@graphql-codegen/cli';

// Schema source = local SDL file from the trip2g repo.
// No live introspection needed — the schema.graphqls is in this monorepo.
const config: CodegenConfig = {
  schema: '../../internal/graph/schema.graphqls',
  documents: 'src/operations/**/*.graphql',
  generates: {
    'src/generated/graphql.ts': {
      plugins: ['typescript', 'typescript-operations', 'typed-document-node'],
      config: {
        useTypeImports: true,
        // Custom scalars in the trip2g schema → TS types.
        scalars: {
          Int64: 'number',
          Time: 'string',
          Upload: 'unknown',
        },
      },
    },
  },
};

export default config;
