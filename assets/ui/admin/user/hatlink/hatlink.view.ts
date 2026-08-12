namespace $.$$ {
	export class $trip2g_admin_user_hatlink extends $.$trip2g_admin_user_hatlink {
		override sub() {
			if (this.hat_url() === '') {
				return [this.Generate(), this.Result()]
			}

			return [this.Link(), this.Result()]
		}

		override generate_allowed(): boolean {
			return this.user_email() !== ''
		}

		override generate() {
			const res = $trip2g_admin_user_hatlink_create({
				input: {
					email: this.user_email(),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateHatLinkPayload') {
				const expires = new $mol_time_moment(res.admin.data.expiresAt)
				this.result('')
				this.hat_expires_at(expires.toString('YYYY-MM-DD hh:mm:ss'))
				this.hat_url(res.admin.data.url)
				return
			}

			this.result('Unexpected response type')
		}

		override hide() {
			this.hat_url('')
			this.hat_expires_at('')
			this.result('')
		}
	}
}
