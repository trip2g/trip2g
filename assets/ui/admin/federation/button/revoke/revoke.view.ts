namespace $.$$ {
	export class $trip2g_admin_federation_button_revoke extends $.$trip2g_admin_federation_button_revoke {
		override handle_click() {
			if (this.id() === 0) {
				throw new Error('Secret ID is not set')
			}

			const res = $trip2g_admin_federation_button_revoke_save({ id: this.id() })

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}
		}
	}
}
