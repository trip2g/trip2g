namespace $.$$ {
	export class $trip2g_admin_user_personaltoken extends $.$trip2g_admin_user_personaltoken {
		// The trigger button lives in the page tools, so the body shows only the result.
		override sub() {
			if (this.token() === '') {
				return [this.Result()]
			}

			return [this.Token(), this.Result()]
		}

		override generate_allowed(): boolean {
			return this.user_id() > 0
		}

		override generate() {
			const res = $trip2g_admin_user_personaltoken_create({
				input: {
					userId: this.user_id(),
					name: 'Issued by admin',
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateUserTokenPayload') {
				this.result('')
				this.token(res.admin.data.plaintextToken)
				this.instructions(res.admin.data.instructions)
				this.expires_at(
					res.admin.data.token.expiresAt
						? new $mol_time_moment(res.admin.data.token.expiresAt).toString('YYYY-MM-DD hh:mm:ss')
						: 'never',
				)
				return
			}

			this.result('Unexpected response type')
		}

		override hide() {
			this.token('')
			this.instructions('')
			this.expires_at('')
			this.result('')
		}
	}
}
