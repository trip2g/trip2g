namespace $.$$ {
	export class $trip2g_user_space extends $.$trip2g_user_space {
		viewer() {
			return this.$.$trip2g_auth_viewer.current()
		}

		override title() {
			return $trip2g_settings.title()
		}

		override open_button() {
			const viewer = this.viewer()
			return viewer.user ? this.OpenButton() : this.SignInButton()
		}

		override editor() {
			const viewer = this.viewer()
		
			if (viewer.role === Role.ADMIN) {
				return this.Editor()
			}
		
			return null
		}

		override admin_link() {
			const viewer = this.viewer()

			if (viewer.role === Role.ADMIN) {
				return this.AdminLink()
			}

			return null
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
			// The AuthWrapper instance is long-lived, so its entered_email/code_sent
			// would otherwise persist across close/reopen and skip straight to the
			// code step. Reset to a clean email prompt every time the dialog opens.
			this.AuthWrapper().back()
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

	export class $trip2g_user_space_nav extends $.$trip2g_user_space_nav {
		// A token's own screen and the new-token form are routes, not sections:
		// they are reachable by link and by URL, and listing them in the menu
		// beside "My account" would make two of the three entries transient.
		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter(id => id !== 'token' && id !== 'token_new')
		}
	}
}
