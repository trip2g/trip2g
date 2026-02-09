namespace $.$$ {
	export class $trip2g_editor_content extends $.$trip2g_editor_content {
		static tiptap(): any {
			return $mol_import.script('/assets/tiptap/tiptap.js').$trip2g_tiptap_bundle
		}

		@ $mol_mem
		tiptap() {
			const t = $trip2g_editor_content.tiptap()
			t.destructor = () => {
				this._editor?.destroy()
				this._editor = null
			}
			return t
		}

		_editor: any = null

		override render() {
			const node = this.dom_node_actual()
			const tiptap = this.tiptap()

			if (!this._editor) {
				this._editor = tiptap.createTiptap()
				this._editor.onChange((md: string) => {
					this.content(md)
				})
				$mol_wire_sync(this._editor).create(node, this.content())
			}
		}

	}
}
