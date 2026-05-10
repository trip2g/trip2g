namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation EnableApiKey($input: EnableApiKeyInput!) {
			admin {
				data: enableApiKey(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on EnableApiKeyPayload {
						apiKey {
							id
						}
					}
				}
			}
		}
	`)
	export class $trip2g_admin_apikey_button_enable extends $.$trip2g_admin_apikey_button_enable {
		override handle_click() {
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
