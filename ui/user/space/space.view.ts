namespace $.$$ {
	export class $trip2g_user_space extends $.$trip2g_user_space {
		viewer() {
			return this.$.$trip2g_auth_viewer.current()
		}

		open_status(opened?: boolean): string | null {
			const KEY = 'userspace'

			if (opened !== undefined) {
				const newVal = opened ? 'open' : null
				this.$.$mol_state_arg.value(KEY, newVal)
				return newVal
			}

			return this.$.$mol_state_arg.value(KEY) ? 'open' : null
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
			return this.viewer().user?.email || '???'
		}
	}
}
