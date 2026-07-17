namespace $.$$ {
	export class $trip2g_admin_gittoken_button_disable extends $.$trip2g_admin_gittoken_button_disable {
		override handle_click() {
			if (this.id() === 0) {
				throw new Error('Git token ID is not set')
			}

			const res = $trip2g_admin_gittoken_button_disable_disable({
				input: {
					id: this.id()
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}
		}
	}
}
