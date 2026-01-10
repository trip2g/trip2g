namespace $.$$ {
	export class $trip2g_admin_layout_editor extends $.$trip2g_admin_layout_editor {

		@ $mol_mem
		override preview_html(): string {
			try {
				const layout = JSON.parse(this.layout_json())
				const res = $mol_fetch.json('/_system/layouts/render', {
					method: 'POST',
					credentials: 'include',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						note_path: this.note_path(),
						layout: layout,
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

	export class $trip2g_admin_layout_editor_preview extends $.$trip2g_admin_layout_editor_preview {
		render() {
			super.render()
			this.dom_node_actual().innerHTML = this.html()
		}
	}
}
