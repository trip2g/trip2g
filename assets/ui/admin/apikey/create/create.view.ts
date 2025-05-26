namespace $.$$ {
	export class $trip2g_admin_apikey_create extends $.$trip2g_admin_apikey_create {
		override body() {
			if (this.api_key() !== '') {
				return [this.ApiKeyView()]
			}

			return super.body()
		}

		override submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateApiKey($input: CreateApiKeyInput!) {
						admin {
							data: createApiKey(input: $input) {
								... on ErrorPayload {
									message
								}
								... on CreateApiKeyPayload {
									value
									apiKey {
										id
									}
								}
							}
						}
					}
				`,
				{
					input: {
						description: this.description(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateApiKeyPayload') {
				this.api_key(res.admin.data.value)
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}