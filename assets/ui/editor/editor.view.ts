namespace $.$$ {
	export class $trip2g_editor extends $.$trip2g_editor {
		dialog_dom() {
			return this.Dialog().dom_node() as HTMLDialogElement
		}

		_mounted = false

		open_status(opened?: boolean): string | null {
			const KEY = 'editor'

			setTimeout(() => {
				if (this.$.$mol_state_arg.value(KEY) === 'open') {
					this.dialog_dom().showModal()
				} else {
					this.dialog_dom().close()
				}

				this._mounted = true
			}, 10)

			if (opened !== undefined) {
				const newVal = opened ? 'open' : null
				this.$.$mol_state_arg.value(KEY, newVal)
				return newVal
			} else {
				// need to mark that dependency.
				const stateOpened = this.$.$mol_state_arg.value(KEY) === 'open'

				if (this._mounted) {
					if (stateOpened) {
						this.dialog_dom().showModal()
					} else {
						this.dialog_dom().close()
					}
				}
			}

			return opened ? 'open' : null
		}

		modal_node() {
			return this.Dialog().dom_node() as HTMLDialogElement
		}

		open() {
			this.open_status(true)
		}

		override close() {
			this.open_status(false)
		}

		override close_event() {
			this.open_status(false)
		}

		override opened(next?: boolean) {
			if (next !== undefined) {
				this.open_status(next)
			}

			return this.$.$mol_state_arg.value('editor') === 'open'
		}

		override close_click(e: MouseEvent) {
			const r = this.modal_node().getBoundingClientRect()

			if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
				this.modal_node().close()
			}
		}
	}
}
