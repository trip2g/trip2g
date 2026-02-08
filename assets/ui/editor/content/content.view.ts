namespace $.$$ {
	export class $trip2g_editor_content extends $.$trip2g_editor_content {
		static milkdown(): any {
			return $mol_import.script('/assets/milkdown/milkdown.js').$trip2g_milkdown_bundle
		}

		milkdown() {
			return $trip2g_editor_content.milkdown()
		}

		_editor: any = null

		override render() {
			const node = this.dom_node_actual()
			const milkdown = this.milkdown()

			if (!this._editor) {
				this._editor = milkdown.createMilkdown()
				this._editor.create(node, this.content())
				this._editor.onChange((md: string) => {
					this.content(md)
				})
			}
		}

		override destructor() {
			if (this._editor) {
				this._editor.destroy()
				this._editor = null
			}
			super.destructor()
		}
	}
}
