import { Node, mergeAttributes, nodeInputRule } from '@tiptap/core'

// Inline atom node for wiki-links: [[page]] or [[page|alias]].
export const WikiLink = Node.create({
	name: 'wikilink',
	group: 'inline',
	inline: true,
	atom: true,

	addAttributes() {
		return {
			href: { default: '' },
			label: { default: null },
		}
	},

	parseHTML() {
		return [{ tag: 'a.tiptap-wikilink' }]
	},

	renderHTML({ HTMLAttributes }) {
		const href = HTMLAttributes.href || ''
		const label = HTMLAttributes.label || href.replace(/^\//, '')
		return [
			'a',
			mergeAttributes({ class: 'tiptap-wikilink', href }, HTMLAttributes),
			label,
		]
	},

	addInputRules() {
		return [
			nodeInputRule({
				find: /\[\[([^\]|]+)(?:\|([^\]]+))?\]\]$/,
				type: this.type,
				getAttributes: (match) => ({
					href: '/' + match[1].trim(),
					label: match[2]?.trim() || null,
				}),
			}),
		]
	},
})
