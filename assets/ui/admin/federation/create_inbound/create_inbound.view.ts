namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreateInboundFederationSecret($input: CreateInboundFederationSecretInput!) {
			admin {
				data: createInboundFederationSecret(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on CreateInboundFederationSecretPayload {
						id
						kid
						secretHex
					}
				}
			}
		}
	`)

	export class $trip2g_admin_federation_create_inbound extends $.$trip2g_admin_federation_create_inbound {
		override body() {
			if (this.secret_hex() !== '') {
				return [this.SecretHexView()]
			}

			return super.body()
		}

		override submit() {
			const res = mutate({
				input: {
					kid: this.kid(),
					description: this.description() || null,
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateInboundFederationSecretPayload') {
				this.secret_hex( res.admin.data.secretHex )
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}
