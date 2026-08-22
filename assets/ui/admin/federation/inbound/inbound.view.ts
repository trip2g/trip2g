namespace $.$$ {
	export class $trip2g_admin_federation_inbound extends $.$trip2g_admin_federation_inbound {
		override body() {
			if (this.secret_hex() !== '') {
				return [this.SecretHexView()]
			}

			return super.body()
		}

		override submit() {
			const res = $trip2g_admin_federation_inbound_create({
				input: {
					kid: this.kid(),
					description: this.description() || null,
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateInboundFederationSecretPayload') {
				this.handover_key( res.admin.data.key )
				this.secret_hex( res.admin.data.secretHex )
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}
