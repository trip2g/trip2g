namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation DisableApiKey($input: DisableApiKeyInput!) {
			admin {
				data: disableApiKey(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on DisableApiKeyPayload {
						apiKey {
							id
						}
					}
				}
			}
		}
	`)
	export class $trip2g_admin_apikey_button_disable extends $.$trip2g_admin_apikey_button_disable {
		override handle_click() {
			console.log('handle click')
			if (this.id() === 0) {
				throw new Error('API key ID is not set')
			}

			const res = mutate({
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