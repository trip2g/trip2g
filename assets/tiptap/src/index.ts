import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import { SlashMenu } from './slash-menu'
import { createMarkdownParser, createMarkdownSerializer } from './markdown'
import type { MarkdownParser, MarkdownSerializer } from 'prosemirror-markdown'

import './style.css'

export interface TiptapInstance {
	create(root: HTMLElement, defaultValue?: string): Promise<void>
	destroy(): Promise<void>
	getMarkdown(): string
	setMarkdown(markdown: string): void
	onChange(callback: (markdown: string) => void): void
}

export function createTiptap(): TiptapInstance {
	let editor: Editor | null = null
	let changeCallback: ((markdown: string) => void) | null = null
	let mdParser: MarkdownParser | null = null
	let mdSerializer: MarkdownSerializer | null = null

	return {
		async create(root: HTMLElement, defaultValue = '') {
			editor = new Editor({
				element: root,
				extensions: [
					StarterKit,
					Placeholder.configure({
						placeholder: 'Type / for commands...',
					}),
					Link.configure({
						openOnClick: false,
						HTMLAttributes: { class: 'tiptap-link' },
					}),
					TaskList,
					TaskItem.configure({ nested: true }),
					SlashMenu,
				],
				editorProps: {
					attributes: { class: 'tiptap-editor' },
				},
				onUpdate: ({ editor: e }) => {
					if (changeCallback && mdSerializer) {
						changeCallback(mdSerializer.serialize(e.state.doc))
					}
				},
			})

			mdParser = createMarkdownParser(editor.schema)
			mdSerializer = createMarkdownSerializer(editor.schema)

			if (defaultValue) {
				const doc = mdParser.parse(defaultValue)
				if (doc) {
					editor.commands.setContent(doc.toJSON())
				}
			}
		},

		async destroy() {
			if (editor) {
				editor.destroy()
				editor = null
				mdParser = null
				mdSerializer = null
			}
		},

		getMarkdown(): string {
			if (!editor || !mdSerializer) return ''
			return mdSerializer.serialize(editor.state.doc)
		},

		setMarkdown(markdown: string) {
			if (!editor || !mdParser) return
			const doc = mdParser.parse(markdown)
			if (doc) {
				editor.commands.setContent(doc.toJSON())
			}
		},

		onChange(callback: (markdown: string) => void) {
			changeCallback = callback
		},
	}
}
