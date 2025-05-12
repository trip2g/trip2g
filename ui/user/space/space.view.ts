namespace $.$$ {
	export class $trip2g_user_space extends $.$trip2g_user_space {
		viewer() {
			return this.$.$trip2g_auth_viewer.current()
		}

		mounted = false

		sub() {
			if (!this.mounted) {
				this.mounted = true
				console.log('set theme')
				this.$.$mol_lights(true)
			}

			return super.sub()
		}

		override open_title(): string {
			const viewer = this.viewer()

			return viewer.user ? 'Личный кабинет' : 'Sign in'
		}

		dialog_dom() {
			return this.Dialog().dom_node() as HTMLDialogElement
		}

		open_status(opened?: boolean): string | null {
			const KEY = 'userspace'

			if (opened) {
				this.dialog_dom().showModal()
			} else {
				this.dialog_dom().close()
			}

			if (opened !== undefined) {
				const newVal = opened ? 'open' : null
				this.$.$mol_state_arg.value(KEY, newVal)
				return newVal
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

		user_email() {
			return this.viewer().user?.email || ''
		}

		override close_click(e: MouseEvent) {
			const r = this.modal_node().getBoundingClientRect()

			if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
				this.modal_node().close()
			}
		}
	}
}
