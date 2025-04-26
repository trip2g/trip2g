const { print, stripIgnoredCharacters } = require('graphql')
const { pascalCase } = require('change-case')
const crypto = require('crypto')

function queryHash(val) {
  const hash = crypto.createHash('sha256')
  hash.update(val)
  return hash.digest('hex')
}

module.exports.plugin = (schema, documents, config) => {
	const operations = []

	const lines = []
  const hashes = {};
	const molPrefix = config.molPrefix || 'change_in_the_config'

	for (const doc of documents) {
		if (!doc.document) {
			continue
		}

		for (const def of doc.document.definitions) {
			if (def.kind !== 'OperationDefinition' || !def.name) continue

			let prefix = 'Query'
			if (def.operation === 'mutation') {
				prefix = 'Mutation'
			}

      hashes[def.name.value] = queryHash(doc.rawSDL)

			operations.push({
				source: doc.rawSDL,
				variablesType: `${def.name.value}${prefix}Variables`,
				resultType: `${def.name.value}${prefix}`,
				hasVars: (def.variableDefinitions?.length || 0) > 0,
			})
		}
	}

	for (const op of operations) {
		const lit = op.source.replace(/\n/g, '\\n').replace(/\t/g, '\\t')

		let vars = ''
		let passVars = ''

		if (op.hasVars) {
			vars = `, variables: ${op.variablesType}`
			passVars = `, variables`
		}

		lines.push(`export function ${molPrefix}_request(query: '${lit}'${vars}): ${op.resultType}`)
	}

	lines.push(
		`export function ${molPrefix}_request(query: any, variables?: any) { return ${molPrefix}_raw_request(query, variables); }`
	)

  lines.push(`export const ${molPrefix}_persist_queries = ${JSON.stringify(hashes)}`)

	return lines.join('\n\n') + '\n\n}'
}
