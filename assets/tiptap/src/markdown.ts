import { defaultMarkdownSerializer, defaultMarkdownParser, MarkdownSerializer, MarkdownParser } from 'prosemirror-markdown'
import MarkdownIt from 'markdown-it'
import type { Schema } from '@tiptap/pm/model'

// Wiki-link plugin for markdown-it.
function wikiLinkPlugin(md: MarkdownIt) {
	md.inline.ruler.after('link', 'wikilink', (state, silent) => {
		const src = state.src
		const pos = state.pos

		if (src[pos] !== '[' || src[pos + 1] !== '[') return false

		const end = src.indexOf(']]', pos + 2)
		if (end === -1) return false

		if (!silent) {
			const content = src.slice(pos + 2, end)
			const parts = content.split('|')
			const href = parts[0].trim()
			const label = (parts[1] || parts[0]).trim()

			const token = state.push('wikilink', 'a', 0)
			token.attrSet('href', `/${href}`)
			token.attrSet('class', 'wikilink')
			token.content = label
		}

		state.pos = end + 2
		return true
	})

	md.renderer.rules.wikilink = (tokens, idx) => {
		const token = tokens[idx]
		const href = token.attrGet('href') || ''
		const cls = token.attrGet('class') || ''
		return `<a href="${href}" class="${cls}">${token.content}</a>`
	}
}

const md = MarkdownIt('commonmark', { html: false }).use(wikiLinkPlugin)

export function createMarkdownParser(schema: Schema): MarkdownParser {
	const tokens: Record<string, any> = {
		...defaultMarkdownParser.tokens,
		// task_list and task_item handled by tiptap extensions natively.
	}

	if (schema.nodes.wikilink) {
		tokens.wikilink = {
			node: 'wikilink',
			getAttrs: (token: any) => ({
				href: token.attrGet('href') || '',
				label: token.content || null,
			}),
		}
	}

	return new MarkdownParser(schema, md, tokens)
}

export function createMarkdownSerializer(schema: Schema): MarkdownSerializer {
	const nodes = { ...defaultMarkdownSerializer.nodes }
	const marks = { ...defaultMarkdownSerializer.marks }

	// Task list serialization.
	if (schema.nodes.taskList) {
		nodes.taskList = (state, node) => {
			state.renderList(node, '  ', () => '')
		}
	}

	if (schema.nodes.taskItem) {
		nodes.taskItem = (state, node) => {
			const checked = node.attrs.checked ? '[x] ' : '[ ] '
			state.write(checked)
			state.renderContent(node)
		}
	}

	// Wikilink serialization: [[page]] or [[page|alias]].
	if (schema.nodes.wikilink) {
		nodes.wikilink = (state, node) => {
			const href = (node.attrs.href || '').replace(/^\//, '')
			const label = node.attrs.label
			if (label && label !== href) {
				state.write(`[[${href}|${label}]]`)
			} else {
				state.write(`[[${href}]]`)
			}
		}
	}

	return new MarkdownSerializer(nodes, marks)
}
