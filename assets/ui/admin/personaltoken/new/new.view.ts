namespace $.$$ {
	export class $trip2g_admin_personaltoken_new extends $.$trip2g_admin_personaltoken_new {
		override body() {
			if (this.created() !== '') {
				return [this.Result_view()]
			}

			return super.body()
		}

		// Reached from a user's own token list, the owner is already known, so
		// the field starts filled and the admin does not retype an id they just
		// clicked through.
		@$mol_mem
		override user_id_field(next?: number): number {
			return next ?? this.user_id()
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
			const userId = this.user_id_field()
			if (!userId) {
				this.result('User ID is required')
				return
			}

			const name = this.name().trim()
			if (name === '') {
				this.result('Name is required')
				return
			}

			const res = $trip2g_admin_personaltoken_new_create({
				input: {
					userId,
					name,
					expiresInDays: this.expires_in_days(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateUserTokenPayload') {
				this.result('')
				this.created(res.admin.data.plaintextToken)
				this.instructions(res.admin.data.instructions)
				return
			}

			this.result('Unexpected response type')
		}
	}
}
