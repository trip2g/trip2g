namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation SetApiKeyMcpAdminTools($input: SetApiKeyMcpAdminToolsInput!) {
			admin {
				data: setApiKeyMcpAdminTools(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on SetApiKeyMcpAdminToolsPayload {
						apiKey {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_apikey_button_mcpadmintools extends $.$trip2g_admin_apikey_button_mcpadmintools {
		override label() {
			return this.enabled() ? 'Disable Admin GraphQL' : 'Enable Admin GraphQL'
		}

		override handle_click() {
			if (this.id() === 0) {
				throw new Error('API key ID is not set')
			}

			const res = mutate({
				input: {
					id: this.id(),
					enabled: !this.enabled(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}
		}
	}
}
