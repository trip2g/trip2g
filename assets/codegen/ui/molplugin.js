// graphql-codegen plugin: emits the $mol-style typed seam for one .graphql file.
// Ported from github.com/trip2g/mol_graphql (codegen/molplugin.js).
//
// For an operation file (query/mutation):
//   export function $trip2g_user_favoritenote_notes(opts?): trip2g_user_favoritenote_notesQuery {
//       return $trip2g_gql_request(`<operation + all spread fragment definitions>`, undefined, opts) as ...
//   }
// The result/variables types are baked in by the generator - no reliance on
// byte-for-byte literal matching (the legacy queries.ts overload model).
// Fragment spreads are merged into the sent document at codegen time
// (transitively, by unique fragment name).
//
// Operations are auto-named from the file location: the canonical name is the
// wrapper symbol without the `$` (user/favoritenote/notes.graphql ->
// trip2g_user_favoritenote_notes). Whatever the author wrote - `query { ... }`
// (anonymous) or any name - is overridden to the canonical BEFORE the stock
// plugin runs, so the wrapper symbol, the result type and the name the
// server/APM sees all match the file path 1:1. Fragments are NOT renamed -
// they are spread by name (Relay model) - but a non-canonical fragment name
// gets a non-blocking warning recommending the path-derived one.
//
// For a fragment file:
//   export type $trip2g_note_card_note = trip2g_note_card_noteFragment
//   export type $trip2g_note_card_note_ref  - bare-name opaque ref, usable in .view.tree
//   export function $trip2g_note_card_note_unmask(ref)  - identity, nullability-preserving
//   export function $trip2g_note_card_note_unmask_not_null(ref)  - throws on null ref
//
// It wraps the stock `typescript` / `typescript-operations` plugins and escapes
// every `$` they (or the embedded GraphQL) produce as `\u0024`: the $mol builder
// scans sources for $-prefixed names to build its module graph, and would
// otherwise treat GraphQL variables ($id) and masking keys (' $fragmentRefs')
// as module references (the reason for the stub dirs id/ input/ filter/ ... at
// the mam workspace root today). TS/JS treat $ as the same character, so types
// and runtime strings are unchanged. $-names WE emit ($trip2g_..., doc-comment
// deps) stay unescaped on purpose - those are real module references.
//
// trip2g additions over the demo version:
//   molScalarsName   rename the `Scalars` type (declaration + references) in
//                    every output of the block - lets a second-endpoint block
//                    declare its own scalars next to the main block's
//                    without a duplicate-identifier collision in `namespace $`
//   molStripHelpers  in schema mode, drop the shared helper aliases
//                    (Maybe/InputMaybe/Exact/MakeOptional/MakeMaybe/MakeEmpty/
//                    Incremental) that the main block's schema types already declare
//
// `config.revalidation` picks the invalidation metadata baked into the wrappers:
//   'all' (default)  pass caller opts through - every query subscribes to the
//                    universal marker, every mutation bumps it (trip2g's
//                    existing reset_marker convention)
//   'by_typenames'   walk the operation against the schema and bake in the
//                    static type set: `reads` for a query, `writes` for a
//                    mutation (payload types plus @touches); an empty set falls
//                    back to the universal marker (unknown effect = assume all)
//   'disable'        bake in an empty set: never subscribe, never bump
// `{ revalidate: false }` at the call site opts out in every mode.

const typescriptPlugin = require('@graphql-codegen/typescript')
const operationsPlugin = require('@graphql-codegen/typescript-operations')
const { getNamedType, isObjectType, isAbstractType, print } = require('graphql')

const SUFFIX = { query: 'Query', mutation: 'Mutation', subscription: 'Subscription' }
const REVALIDATION_MODES = ['all', 'by_typenames', 'disable']

module.exports = {
	plugin: async (schema, documents, config, info) => {
		if (config.molMode === 'schema') {
			let out = flatten(await typescriptPlugin.plugin(schema, documents, stockConfig(config), info))
			if (config.molStripHelpers) out = stripSharedHelpers(out)
			return finalize(out, config)
		}

		const runtime = config.molRuntime || '$graphql'
		const fragments = config.molFragments || {}
		const symbol = config.molSymbol
		const revalidation = config.revalidation || 'all'
		if (!REVALIDATION_MODES.includes(revalidation)) {
			throw new Error(`Unknown revalidation mode "${revalidation}": use ${REVALIDATION_MODES.map(mode => `'${mode}'`).join(' | ')}`)
		}

		// rename operations to the canonical before the stock plugin runs, so the
		// generated result/variables types are derived from the canonical name too
		const named = documents.map(doc => renameOperations(doc, symbol.slice(1)))

		const types = await operationsPlugin.plugin(schema, named, stockConfig(config), info)
		const lines = [finalize(flatten(types), config)]

		for (const doc of named) {
			for (const def of doc.document.definitions) {
				if (def.kind === 'OperationDefinition') {
					lines.push(...operationCode(def, doc, { symbol, runtime, fragments, schema, revalidation }))
				} else if (def.kind === 'FragmentDefinition') {
					lines.push(...fragmentCode(def, { symbol, runtime, location: doc.location }))
				}
			}
		}

		return lines.join('\n')
	},
}

// strip our own config keys before handing config to the stock plugins
function stockConfig(config) {
	const {
		molMode, molRuntime, molFragments, molSymbol, molSchemaTypes,
		molPackage, molRoot, molScalarsName, molStripHelpers, revalidation,
		...rest
	} = config
	return rest
}

function flatten(out) {
	if (typeof out === 'string') return out
	return [...(out.prepend || []), out.content || '', ...(out.append || [])].join('\n')
}

// escape stock-plugin output for the $mol dependency scanner, then apply the
// block's Scalars rename (after escaping: a $-prefixed molScalarsName must
// survive as a real module reference)
function finalize(code, config) {
	let out = escapeDollars(code)
	if (config.molScalarsName) out = out.replace(/\bScalars\b/g, config.molScalarsName)
	return out
}

function escapeDollars(code) {
	return code.replace(/\$/g, '\\u0024')
}

// drop the shared helper aliases the main block's schema types already declare
// in `namespace $` (all single-line in stock plugin output)
function stripSharedHelpers(code) {
	return code
		.split('\n')
		.filter(line => !/^export type (Maybe|InputMaybe|Exact|MakeOptional|MakeMaybe|MakeEmpty|Incremental)\b/.test(line))
		.join('\n')
}

// AST-level rename of every operation definition to the path-derived canonical;
// fragment definitions pass through untouched
function renameOperations(doc, name) {
	return {
		...doc,
		document: {
			...doc.document,
			definitions: doc.document.definitions.map(def =>
				def.kind === 'OperationDefinition'
					? { ...def, name: { kind: 'Name', value: name } }
					: def,
			),
		},
	}
}

function operationCode(def, doc, { symbol, runtime, fragments, schema, revalidation }) {
	if (def.operation === 'subscription') {
		throw new Error(
			`${doc.location}: subscriptions are not supported by this codegen yet - ` +
			`keep them on $trip2g_graphql_raw_subscription`,
		)
	}

	const opName = def.name.value
	const resultType = opName + SUFFIX[def.operation]
	const varsType = resultType + 'Variables'

	// merge spread fragments (transitive closure over the global registry);
	// the operation is printed from its (renamed) AST, not the raw source;
	// @touches is a client-side invalidation hint - never sent to the server
	const closure = fragmentClosure(def, fragments, doc.location)
	const merged = [print(def).trim(), ...closure.map(name => fragments[name].source)]
		.join('\n\n')
		.replace(/\s*@touches\([^)]*\)/g, '')

	// typed variables parameter, plus an optional per-call opts (the invalidation
	// escape hatch: `{ revalidate: false }` opts this call out of refetch-on-mutation)
	const varDefs = def.variableDefinitions || []
	const required = varDefs.some(v => v.type.kind === 'NonNullType' && !v.defaultValue)
	const varsParam = varDefs.length === 0 ? '' : `variables${required ? '' : '?'}: ${varsType}`
	const optsParam = 'opts?: { revalidate?: boolean }'
	const param = [varsParam, optsParam].filter(Boolean).join(', ')
	const varsArg = varDefs.length === 0 ? 'undefined' : 'variables'

	const optsArg = optsArgCode(def, { schema, revalidation, fragments, location: doc.location })

	const code = ['']
	if (closure.length) {
		// doc-comment so the $mol builder records dependencies on the fragment
		// modules (it scans $-names, skipping non-doc comments)
		code.push(`/** Spreads fragments: ${closure.map(name => fragments[name].symbol).join(', ')} */`)
	}
	code.push(
		`export function ${symbol}(${param}): ${resultType} {`,
		`\treturn ${runtime}_request(\`${escapeTemplate(merged)}\`, ${varsArg}, ${optsArg}) as ${resultType}`,
		`}`,
	)
	return code
}

// Third wrapper argument per revalidation mode. 'all' passes the caller opts
// through untouched (universal marker). 'disable' pins an empty type set:
// subscribe/bump nothing. 'by_typenames' bakes in the operation's static type
// set: reads for a query, writes (payload types + @touches) for a mutation;
// an empty computed set falls back to the universal marker (unknown effect =
// assume everything). The `...opts` spread keeps `revalidate: false` working
// while callers cannot override the baked-in sets.
function optsArgCode(def, { schema, revalidation, fragments, location }) {
	if (revalidation === 'all') return 'opts'

	const key = def.operation === 'mutation' ? 'writes' : 'reads'
	if (revalidation === 'disable') return `{ ${key}: [], ...opts }`

	const types = new Set(typeSet(def, schema, fragments, location))
	for (const name of touchesTypes(def, schema, location)) types.add(name)
	if (!types.size) return 'opts'
	return `{ ${key}: [${[...types].sort().map(name => `'${name}'`).join(', ')}], ...opts }`
}

// Object types an operation touches: for a query, what it READS; for a
// mutation, what its payload claims it WROTE. A static walk of the selection
// against the schema: fragment spreads resolve through the global registry,
// abstract types expand to every possible concrete type (a safe superset,
// no runtime __typename needed), root operation types are excluded (every
// query would carry Query - zero discrimination).
function typeSet(def, schema, fragments, location) {
	const roots = new Set([schema.getQueryType(), schema.getMutationType(), schema.getSubscriptionType()].filter(Boolean))
	const seen = new Set()
	const spread = new Set()

	const record = type => {
		if (isObjectType(type)) {
			if (!roots.has(type)) seen.add(type.name)
		} else if (isAbstractType(type)) {
			for (const concrete of schema.getPossibleTypes(type)) seen.add(concrete.name)
		}
	}

	const walk = (selectionSet, parentType) => {
		for (const sel of selectionSet.selections) {
			if (sel.kind === 'Field') {
				if (sel.name.value.startsWith('__')) continue // meta fields carry no schema type
				const fieldDef = parentType.getFields()[sel.name.value]
				if (!fieldDef) throw new Error(`${location}: unknown field "${sel.name.value}" on type "${parentType.name}"`)
				const type = getNamedType(fieldDef.type)
				record(type)
				if (sel.selectionSet) walk(sel.selectionSet, type)
			} else if (sel.kind === 'FragmentSpread') {
				const name = sel.name.value
				if (spread.has(name)) continue
				spread.add(name)
				const frag = fragments[name] // existence checked by fragmentClosure before this runs
				walk(frag.node.selectionSet, schema.getType(frag.node.typeCondition.name.value))
			} else if (sel.kind === 'InlineFragment') {
				const cond = sel.typeCondition ? schema.getType(sel.typeCondition.name.value) : parentType
				walk(sel.selectionSet, cond)
			}
		}
	}

	walk(def.selectionSet, def.operation === 'mutation' ? schema.getMutationType() : schema.getQueryType())
	return [...seen]
}

// @touches(types: ["Note", ...]) - the escape hatch for mutations whose side
// effects reach types absent from their payload (urql's additionalTypenames
// analogue). Parsed here, unioned into `writes`, stripped from the sent text.
function touchesTypes(def, schema, location) {
	const dir = (def.directives || []).find(dir => dir.name.value === 'touches')
	if (!dir) return []
	const arg = (dir.arguments || []).find(arg => arg.name.value === 'types')
	if (!arg || arg.value.kind !== 'ListValue') {
		throw new Error(`${location}: @touches expects a list argument: @touches(types: ["TypeName"])`)
	}
	return arg.value.values.map(value => {
		if (value.kind !== 'StringValue' || !schema.getType(value.value)) {
			throw new Error(`${location}: @touches types must name schema types, got ${value.value ?? value.kind}`)
		}
		return value.value
	})
}

function fragmentCode(def, { symbol, runtime, location }) {
	// symmetric to operation auto-naming, but advisory only: renaming a fragment
	// would break its spread sites, so the declared name stays authoritative
	const canonical = symbol.slice(1)
	if (def.name.value !== canonical) {
		console.warn(
			`${location}: fragment "${def.name.value}" does not match its file location - ` +
			`consider renaming it (and its spreads) to "${canonical}"`,
		)
	}

	const fragType = def.name.value + 'Fragment'
	const refType = `${runtime}_ref<${fragType}>`
	return [
		``,
		`/** Data declared by fragment \`${def.name.value}\` - spread it anywhere as \`...${def.name.value}\`. */`,
		`export type ${symbol} = ${fragType}`,
		``,
		`/** Opaque ref to this fragment - a bare name usable where generics don't fit, e.g. a .view.tree property: \`<prop> null ${symbol}_ref\`. */`,
		`export type ${symbol}_ref = ${refType}`,
		``,
		`/**`,
		` * Identity accessor: turns an opaque fragment ref (masked parent data) into the typed fragment fields.`,
		` * Preserves the ref's nullability: a non-null ref yields the fragment, a nullable ref (nullable schema`,
		` * field, null list element) yields a nullable fragment - the compiler forces the null branch.`,
		` */`,
		`export function ${symbol}_unmask(ref: ${refType}): ${fragType}`,
		`export function ${symbol}_unmask(ref: ${refType} | null | undefined): ${fragType} | null | undefined`,
		`export function ${symbol}_unmask(ref: ${refType} | null | undefined): ${fragType} | null | undefined {`,
		`\treturn ref as ${fragType} | null | undefined`,
		`}`,
		``,
		`/** Checked accessor: unmask that throws on a null/undefined ref - the runtime-checked alternative to TS \`!\`. */`,
		`export function ${symbol}_unmask_not_null(ref: ${refType} | null | undefined): ${fragType} {`,
		`\tif (ref == null) throw new Error('null fragment ref for ${def.name.value}')`,
		`\treturn ref as ${fragType}`,
		`}`,
	]
}

function fragmentClosure(def, fragments, location) {
	const seen = new Set()
	const queue = [...collectSpreads(def)]
	while (queue.length) {
		const name = queue.shift()
		if (seen.has(name)) continue
		const frag = fragments[name]
		if (!frag) throw new Error(`${location}: fragment "${name}" is spread but not defined in any .graphql file`)
		seen.add(name)
		queue.push(...frag.spreads)
	}
	return [...seen]
}

function collectSpreads(node, acc = new Set()) {
	if (node.kind === 'FragmentSpread') acc.add(node.name.value)
	for (const key of ['selectionSet', 'selections']) {
		const sub = node[key]
		if (Array.isArray(sub)) sub.forEach(child => collectSpreads(child, acc))
		else if (sub && typeof sub === 'object') collectSpreads(sub, acc)
	}
	return acc
}

// escape for embedding in a template literal; all `$` become `\u0024` so the
// $mol dependency scanner ignores GraphQL variables (also neutralizes `${`)
function escapeTemplate(str) {
	return str.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\$/g, '\\u0024')
}
