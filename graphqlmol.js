const { print, stripIgnoredCharacters, isEnumType, parse, visit } = require('graphql')
const { pascalCase } = require('change-case')
const crypto = require('crypto')

function queryHash(val) {
	const hash = crypto.createHash('sha256')
	hash.update(val)
	return hash.digest('hex')
}

function extractExportTypes(source, operationName, operationType) {
	const exportTypes = []

	try {
		const ast = parse(source)
		const pathStack = []
		const selectionStack = []

		visit(ast, {
			enter(node, key, parent, path) {
				if (node.kind === 'Field') {
					pathStack.push(node.name.value)

					// Track if we're in an array selection
					const isInArray = parent && parent.kind === 'SelectionSet' && 
						parent.selections && parent.selections.length > 1

					selectionStack.push({
						fieldName: node.name.value,
						isArray: false, // determine later from schema if needed
						parent: selectionStack.length > 0 ? selectionStack[selectionStack.length - 1] : null
					})
				}
			},
			leave(node) {
				if (node.kind === 'Field') {
					// Check for @exportType directive
					const exportDirective = node.directives?.find(d => d.name.value === 'exportType')
					if (exportDirective) {
						const nameArg = exportDirective.arguments?.find(arg => arg.name.value === 'name')
						const singleArg = exportDirective.arguments?.find(arg => arg.name.value === 'single')

						const exportName = nameArg?.value?.value || node.name.value
						const isSingle = singleArg?.value?.value === true

						// Build type path
						const typePath = buildTypePath(pathStack.slice(), operationType, operationName, isSingle)

						exportTypes.push({
							fieldName: node.name.value,
							exportName,
							path: [...pathStack],
							typePath,
							operationName,
							operationType,
							isSingle
						})
					}

					pathStack.pop()
					selectionStack.pop()
				}
			}
		})
	} catch (error) {
		console.warn('Failed to parse GraphQL for @exportType extraction:', error.message)
	}

	return exportTypes
}

function extractOperationVariableTypes(operations) {
	const variableTypes = []

	for (const op of operations) {
		if (op.hasVars) {
			// Extract operation name from variablesType (remove suffix like "QueryVariables")
			const operationName = op.variablesType.replace(/(Query|Mutation|Subscription)Variables$/, '')

			variableTypes.push({
				operationName,
				variablesType: op.variablesType
			})
		}
	}

	return variableTypes
}

function buildTypePath(pathArray, operationType, operationName, isSingle = false) {
	if (pathArray.length === 0) return ''

	// Use generated operation type name instead of generic Query/Mutation/Subscription
	let prefix = 'Query'
	switch (operationType) {
		case 'query':
			prefix = 'Query'
			break
		case 'mutation':
			prefix = 'Mutation'
			break
		case 'subscription':
			prefix = 'Subscription'
			break
		default:
			prefix = 'Query'
	}

	const baseType = `${operationName}${prefix}`

	// Build path with NonNullable wrapping for each level
	// Example: NonNullable<NonNullable<Query['admin']>['backgroundQueue']>['jobs'][0]
	let typePath = baseType
	for (const segment of pathArray) {
		typePath = `NonNullable<${typePath}>['${segment}']`
	}

	if (isSingle) {
		// For single: true, add [0] to get the element type
		return `${typePath}[0]`
	} else {
		// For single: false (default), return the array type itself
		return typePath
	}
}

function generateExportTypeDeclarations(exportTypes, molPrefix) {
	return exportTypes.map(({ exportName, typePath, operationName }) => {
		const typeAlias = `${molPrefix}_${operationName}${exportName}`
		return `export type ${typeAlias} = ${typePath}`
	})
}

function generateVariableTypeDeclarations(variableTypes, molPrefix) {
	return variableTypes.map(({ operationName, variablesType }) => {
		const typeAlias = `${molPrefix}_${operationName}Variables`
		return `export type ${typeAlias} = ${variablesType}`
	})
}

module.exports.plugin = (schema, documents, config) => {
	const operations = []

	const hashes = {}
	const molPrefix = config.molPrefix || 'change_in_the_config'

	// extract enums from schema

	const enums = []
	const typeMap = schema.getTypeMap();

	for (const typeName in typeMap) {
		const type = typeMap[typeName]
		if (isEnumType(type) && !typeName.startsWith('__')) {
			enums.push(type)
		}
	}

	const allExportTypes = []

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

			if (def.operation === 'subscription') {
				prefix = 'Subscription'
			}

      console.log('found operation:', def.name.value, prefix)

			// Extract @exportType directives
			const exportTypes = extractExportTypes(doc.rawSDL, def.name.value, def.operation)
			allExportTypes.push(...exportTypes)

			hashes[def.name.value] = queryHash(doc.rawSDL)

			const op = {
				source: doc.rawSDL,
				type: def.operation,
				variablesType: `${def.name.value}${prefix}Variables`,
				resultType: `${def.name.value}${prefix}`,
				hasVars: (def.variableDefinitions?.length || 0) > 0,
			}

			operations.push(op)
		}
	}

	// Extract variable types from operations
	const allVariableTypes = extractOperationVariableTypes(operations)

	const requestLines = []
	const subscriptionLines = []

	for (const op of operations) {
		const lit = op.source.replace(/\n/g, '\\n').replace(/\t/g, '\\t')

		let vars = ''

		if (op.hasVars) {
			vars = `variables: ${op.variablesType}`
		}

		if (op.type === 'subscription') {
			subscriptionLines.push(
				`export function ${molPrefix}_subscription(query: '${lit}'${vars ? ', ' + vars : ''}): $trip2g_sse_host`
			)
		} else {
      //requestLines.push(`export function ${molPrefix}_request(query: '${lit}'${vars}): ${op.resultType}`)
      requestLines.push(`export function ${molPrefix}_request(query: '${lit}'): (${vars}) => ${op.resultType}`)
		}
	}

	requestLines.push(
		`export function ${molPrefix}_request(query: any) { return ${molPrefix}_raw_request(query); }`
	)

	subscriptionLines.push(
		`export function ${molPrefix}_subscription(query: any, variables?: any) { return ${molPrefix}_raw_subscription(query, variables); }`
	)

	requestLines.push(subscriptionLines.join('\n\n') + '\n\n')

	requestLines.push(`export const ${molPrefix}_persist_queries = ${JSON.stringify(hashes)}`)

	function camelToUnderscore(str) {
		return str.replace(/([a-z])([A-Z])/g, '$1_$2').toLowerCase()
	}

	enums.forEach((e) => {
		// export each enum
		requestLines.push(`export const ${molPrefix}_${camelToUnderscore(e.name)} = ${e.name};`)
	})

	// Generate @exportType declarations
	if (allExportTypes.length > 0) {
		const exportTypeDeclarations = generateExportTypeDeclarations(allExportTypes, molPrefix)
		requestLines.push('// Generated @exportType declarations')
		requestLines.push(...exportTypeDeclarations)
	}

	// Generate variable type declarations
	if (allVariableTypes.length > 0) {
		const variableTypeDeclarations = generateVariableTypeDeclarations(allVariableTypes, molPrefix)
		requestLines.push('// Generated variable type declarations')
		requestLines.push(...variableTypeDeclarations)
	}

	return requestLines.join('\n\n') + '\n\n}'
}
