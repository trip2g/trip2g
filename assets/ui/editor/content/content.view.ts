namespace $.$$ {
	export class $trip2g_editor_content extends $.$trip2g_editor_content {
		static milkdown(): any {
			return $mol_import.script('/assets/milkdown/milkdown.js').$trip2g_milkdown_bundle
		}

		@ $mol_mem	
		milkdown() {
			const m = $trip2g_editor_content.milkdown()
			m.destructor = () => {
				this._editor?.destroy()
				this._editor = null
			}
			return m
		}

		_editor: any = null

		override render() {
			const node = this.dom_node_actual()
			const milkdown = this.milkdown()
			console.log('content', this.content())

			if (!this._editor) {
				this._editor = milkdown.createMilkdown()
				this._editor.onChange((md: string) => {
					this.content(md)
				})
				console.log('create editor', this.content(), node)
				$mol_wire_sync(this._editor).create(node, this.content())
			}
		}

	}
}
