import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import { WikiLink } from './wikilink'
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

interface ToolbarButton {
	label: string
	action: (editor: Editor) => void
	isActive?: (editor: Editor) => boolean
}

function createToolbar(editor: Editor): HTMLElement {
	const toolbar = document.createElement('div')
	toolbar.className = 'tiptap-toolbar'

	const groups: (ToolbarButton | 'sep')[][] = [
		[
			{
				label: 'B',
				action: (e) => e.chain().focus().toggleBold().run(),
				isActive: (e) => e.isActive('bold'),
			},
			{
				label: 'I',
				action: (e) => e.chain().focus().toggleItalic().run(),
				isActive: (e) => e.isActive('italic'),
			},
			{
				label: 'S\u0336',
				action: (e) => e.chain().focus().toggleStrike().run(),
				isActive: (e) => e.isActive('strike'),
			},
			{
				label: '</>',
				action: (e) => e.chain().focus().toggleCode().run(),
				isActive: (e) => e.isActive('code'),
			},
		],
		[
			{
				label: 'H1',
				action: (e) => e.chain().focus().toggleHeading({ level: 1 }).run(),
				isActive: (e) => e.isActive('heading', { level: 1 }),
			},
			{
				label: 'H2',
				action: (e) => e.chain().focus().toggleHeading({ level: 2 }).run(),
				isActive: (e) => e.isActive('heading', { level: 2 }),
			},
			{
				label: 'H3',
				action: (e) => e.chain().focus().toggleHeading({ level: 3 }).run(),
				isActive: (e) => e.isActive('heading', { level: 3 }),
			},
		],
		[
			{
				label: '\u2022 List',
				action: (e) => e.chain().focus().toggleBulletList().run(),
				isActive: (e) => e.isActive('bulletList'),
			},
			{
				label: '1. List',
				action: (e) => e.chain().focus().toggleOrderedList().run(),
				isActive: (e) => e.isActive('orderedList'),
			},
			{
				label: '\u2611 List',
				action: (e) => e.chain().focus().toggleTaskList().run(),
				isActive: (e) => e.isActive('taskList'),
			},
			{
				label: '\u201C\u201D',
				action: (e) => e.chain().focus().toggleBlockquote().run(),
				isActive: (e) => e.isActive('blockquote'),
			},
			{
				label: '{ }',
				action: (e) => e.chain().focus().toggleCodeBlock().run(),
				isActive: (e) => e.isActive('codeBlock'),
			},
		],
		[
			{
				label: '\u2014',
				action: (e) => e.chain().focus().setHorizontalRule().run(),
			},
			{
				label: 'Link',
				action: (e) => {
					if (e.isActive('link')) {
						e.chain().focus().unsetLink().run()
						return
					}
					const href = window.prompt('URL:')
					if (href) {
						e.chain().focus().setLink({ href }).run()
					}
				},
				isActive: (e) => e.isActive('link'),
			},
		],
	]

	const buttons: { el: HTMLButtonElement; isActive?: (editor: Editor) => boolean }[] = []

	for (let gi = 0; gi < groups.length; gi++) {
		if (gi > 0) {
			const sep = document.createElement('span')
			sep.className = 'tiptap-toolbar-sep'
			toolbar.appendChild(sep)
		}
		for (const item of groups[gi]) {
			if (item === 'sep') continue
			const btn = document.createElement('button')
			btn.type = 'button'
			btn.className = 'tiptap-toolbar-btn'
			btn.textContent = item.label
			btn.addEventListener('mousedown', (ev) => {
				ev.preventDefault()
				item.action(editor)
			})
			toolbar.appendChild(btn)
			buttons.push({ el: btn, isActive: item.isActive })
		}
	}

	const updateActive = () => {
		for (const { el, isActive } of buttons) {
			if (isActive) {
				el.classList.toggle('is-active', isActive(editor))
			}
		}
	}

	editor.on('selectionUpdate', updateActive)
	editor.on('transaction', updateActive)

	// Initial state.
	updateActive()

	return toolbar
}

export function createTiptap(): TiptapInstance {
	let editor: Editor | null = null
	let changeCallback: ((markdown: string) => void) | null = null
	let mdParser: MarkdownParser | null = null
	let mdSerializer: MarkdownSerializer | null = null
	let toolbarEl: HTMLElement | null = null
	let editorContainer: HTMLElement | null = null

	return {
		async create(root: HTMLElement, defaultValue = '') {
			editorContainer = document.createElement('div')

			editor = new Editor({
				element: editorContainer,
				extensions: [
					StarterKit,
					Placeholder.configure({
						placeholder: 'Type / for commands...',
					}),
					Link.configure({
						openOnClick: false,
						HTMLAttributes: { class: 'tiptap-link' },
					}),
					WikiLink,
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

			toolbarEl = createToolbar(editor)
			root.appendChild(toolbarEl)
			root.appendChild(editorContainer)

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
			if (toolbarEl) {
				toolbarEl.remove()
				toolbarEl = null
			}
			if (editorContainer) {
				editorContainer.remove()
				editorContainer = null
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
