namespace $.$$ {
	export class $trip2g_admin_apikey_button_enable extends $.$trip2g_admin_apikey_button_enable {
		override handle_click() {
			if (this.id() === 0) {
				throw new Error('API key ID is not set')
			}

			const res = $trip2g_admin_apikey_button_enable_enable({
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
