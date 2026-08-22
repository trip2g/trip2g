namespace $.$$ {
	export class $trip2g_admin_federation_outbound extends $.$trip2g_admin_federation_outbound {
		override submit() {
			// The packed key and the three fields answer the same question, so
			// only one is sent. The server would ignore the others anyway; not
			// sending them is what keeps a half-filled form from reading as a
			// disagreement between the two.
			const key = this.handover_key().trim()

			const res = $trip2g_admin_federation_outbound_create({
				input: key ? {
					key,
					description: this.description() || null,
				} : {
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
