namespace $.$$ {
	export class $trip2g_admin_personaltoken_button_revoke extends $.$trip2g_admin_personaltoken_button_revoke {
		override handle_click() {
			if (this.token_id() === '') {
				throw new Error('token id is not set')
			}

			const res = $trip2g_admin_personaltoken_button_revoke_revoke({
				input: {
					id: this.token_id(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			this.revoked(null)
		}
	}
}
