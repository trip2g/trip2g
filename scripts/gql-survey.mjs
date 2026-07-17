// Survey of frontend GraphQL call sites, driving the .graphql-file migration
// (see codegen/ui/graphqlgen.js). Classifies every call to the legacy request
// functions by shape, so migration batches can be sized and the tricky sites
// flagged. Read-only; prints a summary and writes a JSON report.
//
//   node scripts/gql-survey.mjs [--json out.json]

import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const ts = require('typescript')
let parseGraphQL = null
try {
	parseGraphQL = require('graphql').parse
} catch {}

const ROOT = path.resolve(process.argv[1], '..', '..')
const UI = path.join(ROOT, 'assets', 'ui')

const TARGETS = new Set([
	'$trip2g_graphql_request',
	'$trip2g_graphql_subscription',
	'$trip2g_graphql_raw_request',
	'$trip2g_graphql_raw_subscription',
	'$trip2g_codellm_graphql_raw_request',
])

const SKIP = /\/(queries\.ts|.*\.graphql\.ts)$|\/graphql\/index\.ts$|\/gql\/gql\.ts$/

const sites = []

for (const file of walk(UI)) {
	if (!file.endsWith('.ts') || SKIP.test(file)) continue
	const text = fs.readFileSync(file, 'utf8')
	if (![...TARGETS].some(name => text.includes(name))) continue
	const src = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true)
	visit(src, src, file, text)
}

function walk(dir, acc = []) {
	for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
		const p = path.join(dir, entry.name)
		if (entry.isDirectory()) walk(p, acc)
		else acc.push(p)
	}
	return acc
}

function visit(node, src, file, text) {
	if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && TARGETS.has(node.expression.text)) {
		sites.push(classify(node, src, file, text))
	}
	ts.forEachChild(node, child => visit(child, src, file, text))
}

function classify(call, src, file, text) {
	const fn = call.expression.text
	const rel = path.relative(ROOT, file)
	const line = src.getLineAndCharacterOfLineNumbers?.(call.getStart())
		?? src.getLineAndCharacterOfPosition(call.getStart())
	const arg = call.arguments[0]

	const site = {
		file: rel,
		line: line.line + 1,
		fn,
		endpoint: fn.includes('codellm') ? 'codellm' : 'main',
		arg_shape: 'other',
		op: null,
		op_name: null,
		has_vars: false,
		defs_in_literal: null,
		export_type: false,
		usage: 'other',
		const_name: null,
		reset_cache_calls: 0,
		flags: [],
	}

	if (arg && ts.isNoSubstitutionTemplateLiteral(arg)) {
		site.arg_shape = 'template-literal'
		inspectQuery(site, arg.text)
	} else if (arg && ts.isTemplateExpression(arg)) {
		site.arg_shape = 'template-with-substitution'
		site.flags.push('dynamic query text - manual migration')
	} else if (arg && ts.isStringLiteral(arg)) {
		site.arg_shape = 'string-literal'
		inspectQuery(site, arg.text)
	} else if (arg && ts.isIdentifier(arg)) {
		site.arg_shape = 'identifier'
		// resolve a same-file `const NAME = \`...\`` indirection
		const decl = findConstText(src, arg.text)
		if (decl != null) inspectQuery(site, decl)
		else site.flags.push(`query passed as identifier ${arg.text} - manual`)
	}

	if (site.op === 'subscription' || fn.includes('subscription')) {
		site.op = 'subscription'
		site.flags.push('subscription - out of codegen scope, keep raw')
	}

	// usage shape: stored const (curried style) vs immediately invoked
	const parent = call.parent
	if (ts.isCallExpression(parent) && parent.expression === call) {
		site.usage = 'immediate-invoke'
	} else if (ts.isVariableDeclaration(parent) && ts.isIdentifier(parent.name)) {
		site.usage = 'stored-const'
		site.const_name = parent.name.text
		// second-arg opts ({ resetCache }) at any use of the stored const
		const uses = text.split(parent.name.text).length - 1
		site.const_uses = uses - 1
		const re = new RegExp(escapeRe(parent.name.text) + String.raw`\s*\(`, 'g')
		let m
		while ((m = re.exec(text))) {
			const tail = text.slice(m.index, m.index + 600)
			if (/resetCache/.test(tail.slice(0, tail.indexOf(')') + 1 || 600))) site.reset_cache_calls++
		}
	}

	if (site.export_type) site.flags.push('@exportType - manual migration (fragment or hand type)')
	if (site.defs_in_literal != null && site.defs_in_literal > 1) site.flags.push('multiple definitions in one literal - split into files')

	return site
}

function inspectQuery(site, query) {
	site.export_type = /@exportType/.test(query)
	const cleaned = query.replace(/@exportType\s*(\([^)]*\))?\s*/g, '')
	if (parseGraphQL) {
		try {
			const doc = parseGraphQL(cleaned)
			const ops = doc.definitions.filter(def => def.kind === 'OperationDefinition')
			site.defs_in_literal = doc.definitions.length
			if (ops[0]) {
				site.op = ops[0].operation
				site.op_name = ops[0].name?.value ?? null
				site.has_vars = (ops[0].variableDefinitions?.length ?? 0) > 0
			}
			return
		} catch (err) {
			site.flags.push('graphql parse failed: ' + String(err.message).slice(0, 80))
		}
	}
	site.op = /^\s*mutation\b/m.test(cleaned) ? 'mutation' : /^\s*subscription\b/m.test(cleaned) ? 'subscription' : 'query'
	site.has_vars = /\$\w+\s*:/.test(cleaned)
}

function findConstText(src, name) {
	let found = null
	const scan = node => {
		if (found != null) return
		if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.name.text === name && node.initializer) {
			if (ts.isNoSubstitutionTemplateLiteral(node.initializer) || ts.isStringLiteral(node.initializer)) {
				found = node.initializer.text
			}
		}
		ts.forEachChild(node, scan)
	}
	scan(src)
	return found
}

function escapeRe(str) {
	return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// ---- report ----

const byShape = {}
for (const site of sites) {
	const shape = [
		site.op ?? '?',
		site.arg_shape,
		site.usage,
		site.has_vars ? 'vars' : 'no-vars',
	].join(' / ')
	;(byShape[shape] ??= []).push(site)
}

const byDir = {}
for (const site of sites) {
	const segs = site.file.replace(/^assets\/ui\//, '').split('/')
	const dir = segs[0] === 'admin' ? segs.slice(0, 2).join('/') : segs[0]
	const entry = (byDir[dir] ??= { sites: 0, files: new Set(), flagged: 0 })
	entry.sites++
	entry.files.add(site.file)
	if (site.flags.length) entry.flagged++
}

console.log(`total call sites: ${sites.length} in ${new Set(sites.map(s => s.file)).size} files\n`)
console.log('== shapes ==')
for (const [shape, list] of Object.entries(byShape).sort((a, b) => b[1].length - a[1].length)) {
	console.log(String(list.length).padStart(5), ' ', shape)
}
console.log('\n== flagged sites (manual attention) ==')
for (const site of sites.filter(s => s.flags.length)) {
	console.log(`${site.file}:${site.line}  [${site.fn}]  ${site.flags.join('; ')}`)
}
console.log('\n== resetCache users (stored consts called with opts) ==')
for (const site of sites.filter(s => s.reset_cache_calls > 0)) {
	console.log(`${site.file}:${site.line}  const ${site.const_name}  resetCache calls: ${site.reset_cache_calls}`)
}
console.log('\n== per-directory (batching) ==')
for (const [dir, entry] of Object.entries(byDir).sort((a, b) => b[1].sites - a[1].sites)) {
	console.log(String(entry.sites).padStart(5), String(entry.files.size).padStart(4) + 'f', entry.flagged ? `${String(entry.flagged).padStart(3)} flagged` : '   ', ' ', dir)
}

const jsonIdx = process.argv.indexOf('--json')
if (jsonIdx > -1 && process.argv[jsonIdx + 1]) {
	fs.writeFileSync(process.argv[jsonIdx + 1], JSON.stringify(sites, null, '\t'))
	console.log(`\nwrote ${process.argv[jsonIdx + 1]}`)
}
