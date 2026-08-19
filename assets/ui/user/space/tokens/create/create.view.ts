namespace $.$$ {
	export class $trip2g_user_space_tokens_create extends $.$trip2g_user_space_tokens_create {
		// Once the token exists the form is gone: it is shown once, and leaving
		// the fields under it invites a second one nobody asked for.
		override body() {
			if (this.created() !== '') {
				return [this.Result_view()]
			}

			return super.body()
		}

		expires_in_days(): number | null {
			switch (this.expiry()) {
				case '30d':
					return 30
				case '90d':
					return 90
				case '1y':
					return 365
				default:
					return null
			}
		}

		override submit() {
			const name = this.name().trim()
			if (name === '') {
				this.result(this.name_required())
				return
			}

			const res = $trip2g_user_space_tokens_create_create({
				input: {
					name,
					expiresInDays: this.expires_in_days(),
				},
			})

			const payload = res.createUserToken

			if (payload.__typename === 'ErrorPayload') {
				this.result(payload.message)
				return
			}

			if (payload.__typename === 'CreateUserTokenPayload') {
				this.result('')
				this.created(payload.plaintextToken)
				this.instructions(payload.instructions)
				return
			}

			this.result('Unexpected response type')
		}
	}
}
