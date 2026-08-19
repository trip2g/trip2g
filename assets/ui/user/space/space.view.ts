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

		// A dialog's backdrop is painted by the dialog element itself, so a click on
		// it targets the dialog while a click on anything inside targets that child.
		// Comparing the target is exact; comparing coordinates against the dialog's
		// rect was not — a click raised by the keyboard carries 0,0, and Enter on a
		// menu link read as a backdrop click and shut the account.
		override close_click(e: MouseEvent) {
			if (e.target !== this.modal_node()) return

			this.modal_node().close()
		}
	}

	export class $trip2g_user_space_nav extends $.$trip2g_user_space_nav {
		// $mol_book2_catalog asks for menu_title only when the spread is itself a
		// book, so a plain page is listed under its own title. A page that wants
		// a short label in the menu and a full one in the header declares both.
		override spread_title(spread: string) {
			const page = this.Spread(spread) as $mol_view & { menu_title?(): string }
			return page.menu_title?.() || super.spread_title(spread)
		}

		// A token's own screen and the new-token form are routes, not sections:
		// they are reachable by link and by URL, and listing them in the menu
		// beside "My account" would make two of the three entries transient.
		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter(id => id !== 'token' && id !== 'token_new')
		}
	}
}
