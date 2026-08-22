namespace $.$$ {
	export class $trip2g_admin_federation_button_rotate extends $.$trip2g_admin_federation_button_rotate {
		override handle_click() {
			if (this.kid() === '') {
				throw new Error('Key ID is not set')
			}

			const res = $trip2g_admin_federation_button_rotate_save({ kid: this.kid() })

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}
		}
	}
}
