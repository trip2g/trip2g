namespace $.$$ {
	type Me = {
		user: {
			id: number
			email: string
			created_at: string
		} | null
	}

	export class $trip2g_auth extends $.$trip2g_auth {
		@$mol_mem
		me(next?: any) {
			if (!next) {
				return this.$.$mol_fetch.json('http://localhost:8081/api/me') as Me
			}

			return next ?? null
		}

		sub() {
			const me = this.me()
			console.log('me', me)
			if (me.user) {
				return [this.AppView()]
			}

			const email = this.$.$mol_state_arg.value('email')
			if (email) {
				return [this.CodeForm()]
			}

			return [this.EmailForm()]
		}
	}

	export class $trip2g_auth_email_form extends $.$trip2g_auth_email_form {
		@$mol_mem
		request_error(next?: string): string {
			return next ?? ''
		}

		email_bid() {
			return this.request_error() || ''
		}

		submit() {
			const url = 'http://localhost:8081/api/requestemailsignin'

			const res = this.$.$mol_fetch.json(url, {
				method: 'post',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email: this.email() }),
			}) as {
				success: boolean
				errors?: null | [string]
				message?: string
			}

			if (res.success) {
				this.$.$mol_state_arg.value('email', this.email())
			} else if (res.errors) {
				this.request_error(res.errors?.join(', ') ?? 'Unknown error')
			} else if (res.message) {
				this.request_error(res.message)
			}
		}
	}

	export class $trip2g_auth_code_form extends $.$trip2g_auth_code_form {
		@$mol_mem
		request_error(next?: string): string {
			return next ?? ''
		}

		code_bid() {
			return this.request_error() || ''
		}

		email() {
			return this.$.$mol_state_arg.value('email')
		}

		submit() {
			const url = 'http://localhost:8081/api/signinbyemail'

			const res = this.$.$mol_fetch.json(url, {
				method: 'post',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					email: this.email(),
					code: parseInt(this.code(), 10),
				}),
			}) as {
				success: boolean
				errors?: null | [string]
				message?: string
			}

			if (res.success) {
				alert('Success')
			} else if (res.errors) {
				this.request_error(res.errors?.join(', ') ?? 'Unknown error')
			} else if (res.message) {
				this.request_error(res.message)
			}
		}
	}
}
