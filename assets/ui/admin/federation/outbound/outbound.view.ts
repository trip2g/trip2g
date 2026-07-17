namespace $.$$ {
	export class $trip2g_admin_federation_outbound extends $.$trip2g_admin_federation_outbound {
		override submit() {
			const res = $trip2g_admin_federation_outbound_create({
				input: {
					kid: this.kid(),
					secretHex: this.secret_hex(),
					kbURL: this.kb_url(),
					description: this.description() || null,
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateOutboundFederationSecretPayload') {
				this.result( `Created: ${ res.admin.data.kid }` )
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}
