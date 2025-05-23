namespace $.$$ {
	export class $trip2g_user_space extends $.$trip2g_user_space {
		viewer() {
			return this.$.$trip2g_auth_viewer.current()
		}

		override open_title(): string {
			const viewer = this.viewer()

			return viewer.user ? 'Личный кабинет' : 'Sign in'
		}

		dialog_dom() {
			return this.Dialog().dom_node() as HTMLDialogElement
		}

		_mounted = false

		open_status(opened?: boolean): string | null {
			const KEY = 'userspace'

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
				// need to mark that dependency
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

		close() {
			this.open_status(false)
		}

		override close_event() {
			this.open_status(false);
		}

		user_email() {
			return this.viewer().user?.email || ''
		}

		override signout() {
			super.signout()
			window.location.reload()
		}

		override reload_page()  {
			window.location.reload()
		}

		override close_click(e: MouseEvent) {
			const r = this.modal_node().getBoundingClientRect()

			if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
				this.modal_node().close()
			}
		}
	}
}
