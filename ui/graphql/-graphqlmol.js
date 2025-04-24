const { print, stripIgnoredCharacters } = require('graphql');
const { pascalCase } = require('change-case');

module.exports.plugin = (schema, documents, config) => {
  const operations = [];

  const lines = [];

  for (const doc of documents) {
      // lines.push(`/* ${JSON.stringify(doc, null, '  ')} */`);
    if (!doc.document) continue;
    for (const def of doc.document.definitions) {
      if (def.kind !== 'OperationDefinition' || !def.name) continue;

      let prefix = 'Query';
      if (def.operation === 'mutation') {
        prefix = 'Mutation';
      }

      operations.push({
        source: doc.rawSDL,
        variablesType: `${def.name.value}${prefix}Variables`,
        resultType: `${def.name.value}${prefix}`,
        hasVars: (def.variableDefinitions?.length || 0) > 0,
      });
    }
  }

  for (const op of operations) {
    const lit = op.source.replace(/\n/g, '\\n').replace(/\t/g, '\\t');

    let vars = '';
    let passVars = '';

    if (op.hasVars) {
      vars = `, variables: ${op.variablesType}`;
      passVars = `, variables`;
    }

    lines.push(`export function $trip2g_graphql_request(query: '${lit}'${vars}): ${op.resultType}`);
  }

  lines.push(`export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }`);

  return lines.join('\n\n') + '\n\n}';

  // return `/* ${JSON.stringify(operations, null, '  ')} */`;
};
