namespace $.$$ {

	type Block = {
		id?: string
		type: string
		name?: string
		args?: Record<string, any>
		expr?: string
		condition?: string
		collection?: string
		iterator?: string
		html?: string
		path?: string
		content?: Block[]
	}

	type Layout = {
		meta: Record<string, any>
		body: Block[]
	}

	const block_types = ['block', 'if', 'range', 'expr', 'html', 'note_content', 'include_note']

	// Simple nanoid-like generator
	const alphabet = '0123456789abcdefghijklmnopqrstuvwxyz'
	function nanoid(size = 12) {
		const bytes = crypto.getRandomValues(new Uint8Array(size))
		let id = ''
		for (let i = 0; i < size; i++) {
			id += alphabet[bytes[i] % alphabet.length]
		}
		return id
	}

	function ensure_block_ids(blocks: Block[]): Block[] {
		return blocks.map(b => ({
			...b,
			id: b.id || nanoid(),
			content: b.content ? ensure_block_ids(b.content) : undefined,
		}))
	}

	export class $trip2g_admin_layout_editor extends $.$trip2g_admin_layout_editor {

		@ $mol_mem
		layout(next?: Layout): Layout {
			if (next !== undefined) {
				return {
					...next,
					body: ensure_block_ids(next.body),
				}
			}
			return {
				meta: {},
				body: ensure_block_ids([{ type: 'note_content' }]),
			}
		}

		@ $mol_mem
		block_rows() {
			return this.layout().body.map((_, index) => this.Block(index))
		}

		@ $mol_mem_key
		Block(index: number) {
			const obj = new $trip2g_admin_layout_editor_block()
			obj.block = () => this.layout().body[index] ?? null
			obj.block_index = () => index
			obj.adopt = transfer => this.block_adopt(transfer)
			obj.receive = block => this.block_receive_before(index, block)
			obj.block_changed = block => this.block_update(index, block)
			return obj
		}

		block_update(index: number, block: Block) {
			const layout = this.layout()
			const body = [...layout.body]
			body[index] = block
			this.layout({ ...layout, body })
		}

		@ $mol_mem
		palette_items() {
			return block_types.map(type => this.Palette_item(type))
		}

		@ $mol_mem_key
		Palette_item(type: string) {
			const obj = new $trip2g_admin_layout_editor_palette_item()
			obj.block_type = () => type
			return obj
		}

		block_adopt(transfer: DataTransfer): Block | null {
			const json = transfer.getData('text/plain')
			if (!json) return null
			try {
				const data = JSON.parse(json) as Block
				// Find existing block by id
				if (data.id) {
					const existing = this.layout().body.find(b => b.id === data.id)
					if (existing) return existing
				}
				// New block from palette - assign id
				return { ...data, id: nanoid() }
			} catch {
				return null
			}
		}

		block_receive_before(index: number, block: Block) {
			const layout = this.layout()
			const body = layout.body.filter(b => b.id !== block.id)
			// Adjust index if block was before target
			const oldIndex = layout.body.findIndex(b => b.id === block.id)
			const adjustedIndex = oldIndex !== -1 && oldIndex < index ? index - 1 : index
			body.splice(adjustedIndex, 0, block)
			this.layout({ ...layout, body })
		}

		block_receive_end(block: Block) {
			const layout = this.layout()
			const body = layout.body.filter(b => b.id !== block.id)
			body.push(block)
			this.layout({ ...layout, body })
		}

		block_trash(block: Block) {
			const layout = this.layout()
			const body = layout.body.filter(b => b.id !== block.id)
			this.layout({ ...layout, body })
		}

		@ $mol_mem
		override preview_html(): string {
			try {
				const res = $mol_fetch.json('/_system/layouts/render', {
					method: 'POST',
					credentials: 'include',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						note_path: this.note_path(),
						layout: this.layout(),
					}),
				}) as { html?: string; error?: string }

				if (res.error) {
					return `<pre style="color:red">${res.error}</pre>`
				}
				return res.html || ''
			} catch (e: any) {
				return `<pre style="color:red">${e.message}</pre>`
			}
		}
	}

	export class $trip2g_admin_layout_editor_block extends $.$trip2g_admin_layout_editor_block {

		@ $mol_mem
		override block_json() {
			return JSON.stringify(this.block())
		}

		@ $mol_mem
		block_type() {
			return this.block()?.type ?? ''
		}

		@ $mol_mem
		block_name() {
			const block = this.block()
			if (!block) return ''
			return block.name || block.expr || block.condition || block.path || ''
		}

		@ $mol_mem
		override wrapper_rows() {
			const rows: $mol_view[] = [this.Header()]
			if (this.expanded()) {
				rows.push(this.Form())
			}
			return rows
		}

		@ $mol_mem
		Header() {
			const obj = new $mol_button_minor()
			obj.title = () => `${this.block_type()} ${this.block_name()}`
			obj.click = () => this.expanded(!this.expanded())
			return obj
		}

		@ $mol_mem
		Form() {
			const obj = new $trip2g_admin_layout_editor_block_form()
			obj.block = () => this.block()
			obj.block_changed = block => this.block_changed(block)
			return obj
		}
	}

	export class $trip2g_admin_layout_editor_block_form extends $.$trip2g_admin_layout_editor_block_form {

		@ $mol_mem
		override fields() {
			const block = this.block()
			if (!block) return []

			const fields: $mol_view[] = []
			const type = block.type

			if (type === 'block') {
				fields.push(this.field_name())
			}
			if (type === 'if') {
				fields.push(this.field_condition())
			}
			if (type === 'range') {
				fields.push(this.field_collection())
				fields.push(this.field_iterator())
			}
			if (type === 'expr') {
				fields.push(this.field_expr())
			}
			if (type === 'html') {
				fields.push(this.field_html())
			}
			if (type === 'include_note') {
				fields.push(this.field_path())
			}

			return fields
		}

		update_block(key: keyof Block, value: string) {
			const block = this.block()
			if (!block) return
			this.block_changed({ ...block, [key]: value })
		}

		@ $mol_mem
		field_name() {
			const obj = new $mol_form_field()
			obj.name = () => 'Name'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('name', next)
					return this.block()?.name ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_condition() {
			const obj = new $mol_form_field()
			obj.name = () => 'Condition'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('condition', next)
					return this.block()?.condition ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_collection() {
			const obj = new $mol_form_field()
			obj.name = () => 'Collection'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('collection', next)
					return this.block()?.collection ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_iterator() {
			const obj = new $mol_form_field()
			obj.name = () => 'Iterator'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('iterator', next)
					return this.block()?.iterator ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_expr() {
			const obj = new $mol_form_field()
			obj.name = () => 'Expression'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('expr', next)
					return this.block()?.expr ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_html() {
			const obj = new $mol_form_field()
			obj.name = () => 'HTML'
			obj.Content = () => {
				const input = new $mol_textarea()
				input.value = next => {
					if (next !== undefined) this.update_block('html', next)
					return this.block()?.html ?? ''
				}
				return input
			}
			return obj
		}

		@ $mol_mem
		field_path() {
			const obj = new $mol_form_field()
			obj.name = () => 'Path'
			obj.Content = () => {
				const input = new $mol_string()
				input.value = next => {
					if (next !== undefined) this.update_block('path', next)
					return this.block()?.path ?? ''
				}
				return input
			}
			return obj
		}
	}

	export class $trip2g_admin_layout_editor_palette_item extends $.$trip2g_admin_layout_editor_palette_item {

		@ $mol_mem
		override block_json() {
			const type = this.block_type()
			const block: Block = { type }

			// Add required fields for certain types
			if (type === 'block') block.name = 'new_block'
			if (type === 'if') block.condition = 'true'
			if (type === 'range') {
				block.collection = 'items'
				block.iterator = 'item'
			}
			if (type === 'expr') block.expr = 'note.Title()'
			if (type === 'html') block.html = '<div></div>'
			if (type === 'include_note') block.path = '/_sidebar.md'

			return JSON.stringify(block)
		}
	}

	export class $trip2g_admin_layout_editor_preview extends $.$trip2g_admin_layout_editor_preview {
		override render() {
			super.render()
			this.dom_node_actual().innerHTML = this.html()
		}
	}
}
