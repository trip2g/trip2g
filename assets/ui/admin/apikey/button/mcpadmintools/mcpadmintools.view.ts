namespace $.$$ {
	export class $trip2g_admin_apikey_button_mcpadmintools extends $.$trip2g_admin_apikey_button_mcpadmintools {
		override label() {
			return this.enabled_state() ? 'Disable Admin GraphQL' : 'Enable Admin GraphQL'
		}

		override handle_click() {
			if (this.id() === 0) {
				throw new Error('API key ID is not set')
			}

			const res = $trip2g_admin_apikey_button_mcpadmintools_mcpadmintools({
				input: {
					id: this.id(),
					enabled: !this.enabled_state(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}
		}
	}
}
