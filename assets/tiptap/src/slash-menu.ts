import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

export interface SlashMenuItem {
	title: string
	description: string
	command: (props: { editor: any; range: { from: number; to: number } }) => void
}

const defaultItems: SlashMenuItem[] = [
	{
		title: 'Heading 1',
		description: 'Large section heading',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).setNode('heading', { level: 1 }).run()
		},
	},
	{
		title: 'Heading 2',
		description: 'Medium section heading',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).setNode('heading', { level: 2 }).run()
		},
	},
	{
		title: 'Heading 3',
		description: 'Small section heading',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).setNode('heading', { level: 3 }).run()
		},
	},
	{
		title: 'Bullet List',
		description: 'Unordered list',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).toggleBulletList().run()
		},
	},
	{
		title: 'Numbered List',
		description: 'Ordered list',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).toggleOrderedList().run()
		},
	},
	{
		title: 'Task List',
		description: 'Checklist with tasks',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).toggleTaskList().run()
		},
	},
	{
		title: 'Blockquote',
		description: 'Quote block',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).toggleBlockquote().run()
		},
	},
	{
		title: 'Code Block',
		description: 'Fenced code block',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).toggleCodeBlock().run()
		},
	},
	{
		title: 'Horizontal Rule',
		description: 'Divider line',
		command: ({ editor, range }) => {
			editor.chain().focus().deleteRange(range).setHorizontalRule().run()
		},
	},
]

function createMenu(items: SlashMenuItem[], editor: any, range: { from: number; to: number }) {
	const menu = document.createElement('div')
	menu.className = 'tiptap-slash-menu'

	items.forEach((item, index) => {
		const btn = document.createElement('button')
		btn.className = 'tiptap-slash-menu-item'
		if (index === 0) btn.classList.add('is-selected')
		btn.innerHTML = `<span class="tiptap-slash-menu-title">${item.title}</span><span class="tiptap-slash-menu-desc">${item.description}</span>`
		btn.addEventListener('mousedown', (e) => {
			e.preventDefault()
			item.command({ editor, range })
		})
		menu.appendChild(btn)
	})

	return menu
}

const pluginKey = new PluginKey('slashMenu')

export const SlashMenu = Extension.create({
	name: 'slashMenu',

	addProseMirrorPlugins() {
		const editor = this.editor
		let menuEl: HTMLElement | null = null
		let selectedIndex = 0
		let filteredItems: SlashMenuItem[] = []
		let slashRange: { from: number; to: number } | null = null

		const destroy = () => {
			menuEl?.remove()
			menuEl = null
			slashRange = null
			filteredItems = []
			selectedIndex = 0
		}

		const updateSelection = () => {
			if (!menuEl) return
			menuEl.querySelectorAll('.tiptap-slash-menu-item').forEach((el, i) => {
				el.classList.toggle('is-selected', i === selectedIndex)
				if (i === selectedIndex) el.scrollIntoView({ block: 'nearest' })
			})
		}

		return [
			new Plugin({
				key: pluginKey,
				props: {
					handleKeyDown(view, event) {
						if (!menuEl) return false

						if (event.key === 'ArrowDown') {
							selectedIndex = (selectedIndex + 1) % filteredItems.length
							updateSelection()
							return true
						}
						if (event.key === 'ArrowUp') {
							selectedIndex = (selectedIndex - 1 + filteredItems.length) % filteredItems.length
							updateSelection()
							return true
						}
						if (event.key === 'Enter') {
							const item = filteredItems[selectedIndex]
							if (item && slashRange) {
								item.command({ editor, range: slashRange })
								destroy()
							}
							return true
						}
						if (event.key === 'Escape') {
							destroy()
							return true
						}

						return false
					},

					decorations(state) {
						return pluginKey.getState(state)
					},
				},

				state: {
					init: () => DecorationSet.empty,
					apply(tr, _oldState, _oldEditorState, newState) {
						const { selection } = newState
						const { $from } = selection

						if (!selection.empty) {
							destroy()
							return DecorationSet.empty
						}

						const textBefore = $from.parent.textContent.slice(0, $from.parentOffset)
						const slashMatch = textBefore.match(/\/(\w*)$/)

						if (!slashMatch) {
							destroy()
							return DecorationSet.empty
						}

						const query = slashMatch[1].toLowerCase()
						const from = $from.pos - slashMatch[0].length
						const to = $from.pos

						slashRange = { from, to }
						filteredItems = defaultItems.filter(
							(item) =>
								item.title.toLowerCase().includes(query) ||
								item.description.toLowerCase().includes(query)
						)

						if (filteredItems.length === 0) {
							destroy()
							return DecorationSet.empty
						}

						selectedIndex = Math.min(selectedIndex, filteredItems.length - 1)

						// Position menu near cursor.
						const coords = editor.view.coordsAtPos(from)
						if (menuEl) menuEl.remove()
						menuEl = createMenu(filteredItems, editor, slashRange)
						menuEl.style.top = `${coords.bottom + 4}px`
						menuEl.style.left = `${coords.left}px`
						document.body.appendChild(menuEl)
						updateSelection()

						return DecorationSet.empty
					},
				},
			}),
		]
	},
})
