const { print } = require('graphql')
const { Kind } = require('graphql')

function snakeCase(str) {
  return str
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')  // camelCase → camel_Case
    .replace(/[-\s]+/g, '_')                // пробелы/дефисы → _
    .toLowerCase()
}

module.exports = {
  plugin: (schema, documents) => {
    const allOps = documents
      .flatMap(doc => doc.document.definitions)
      .filter(def => def.kind === Kind.OPERATION_DEFINITION && def.operation === 'query')

    const output = allOps.map(op => {
      const name = op.name?.value
      if (!name) return ''
      const snake = snakeCase(name)
      const query = print(op).replace(/`/g, '\\`') // экранируем `
      const returnType = `${name}Query`;
      return `\texport const $trip2g_graphql_${snake} = (variables: ${returnType}Variables) =>
\t\t$trip2g_graphql_request<${returnType}>(\`${query}\`, variables)`;
    })

    return output.join('\n\n') + '\n\n}';
  }
}
