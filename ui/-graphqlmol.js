const { print, stripIgnoredCharacters } = require('graphql')
const { pascalCase } = require('change-case')
const crypto = require('crypto')

function queryHash(val) {
	const hash = crypto.createHash('sha256')
	hash.update(injectTypename(val))
	return hash.digest('hex')
}

function injectTypename(source) {
	let out = ''
	let i = 0
	const n = source.length

	// Текущее состояние конечного автомата
	let state = 'default' // 'default' | 'string' | 'blockString' | 'comment'
	let quoteChar = null // '"' или "'"
	let blockQuoteLen = 0 // 1 (обычная строка) или 3 (блок-строка)

	while (i < n) {
		const c = source[i]
		const next3 = source.slice(i, i + 3)

		/* ---------- переходы состояний ---------- */
		if (state === 'default') {
			if (c === '#') {
				// начало комментария
				state = 'comment'
				out += c
				i++
				continue
			}

			if (next3 === '"""' || next3 === "'''") {
				// начало блок-строки
				state = 'blockString'
				quoteChar = c
				blockQuoteLen = 3
				out += next3
				i += 3
				continue
			}

			if (c === '"' || c === "'") {
				// начало обычной строки
				state = 'string'
				quoteChar = c
				blockQuoteLen = 1
				out += c
				i++
				continue
			}

			if (c === '{') {
				// точка инъекции
				out += c
				i++

				// пропускаем пробелы, перевод строки, запятые
				let j = i
				while (j < n && /[\s,\r\n]/.test(source[j])) j++

				const hasTypename = source.startsWith('__typename', j)
				if (!hasTypename) out += ' __typename'
				continue
			}
		} else if (state === 'comment') {
			out += c
			i++
			if (c === '\n') state = 'default'
			continue
		} else if (state === 'string') {
			out += c
			i++
			// выходим, если встретили неэкранированную закрывающую кавычку
			if (c === quoteChar && source[i - 2] !== '\\') state = 'default'
			continue
		} else if (state === 'blockString') {
			out += c
			i++
			// проверяем последние 3 символа на тройные кавычки
			if (source.slice(i - blockQuoteLen, i) === quoteChar.repeat(blockQuoteLen)) state = 'default'
			continue
		}

		// обычный проход
		out += c
		i++
	}

	return out
}

module.exports.plugin = (schema, documents, config) => {
	const operations = []

	const lines = []
	const hashes = {}
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
