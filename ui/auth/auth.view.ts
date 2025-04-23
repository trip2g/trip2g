namespace $.$$ {
	const query = $trip2g_graphql(/* GraphQL */ `
		mutation RequestCode($input: RequestEmailSignInCodeInput!) {
			requestEmailSignInCode(input: $input) {
				... on ErrorPayload {
					__typename
					message
				}
				... on RequestEmailSignInCodePayload {
					__typename
					success
				}
			}
		}
	`)
	console.log(query);

	/* GraphQL */ `
		mutation SignIn($input: SignInByEmailInput!) {
			signInByEmail(input: $input) {
				... on SignInPayload {
					__typename
					token
				}
				... on ErrorPayload {
					__typename
					message
				}
			}
		}

		query Viewer {
			viewer {
				user {
					id
					email
					createdAt
				}
			}
		}
	`

	export class $trip2g_auth extends $.$trip2g_auth {
		me_request() {
			//const viewer = $trip2g_graphql_viewer()
			//console.log(viewer)

			return this.$.$mol_fetch.json('/api/me', {
				credentials: 'include',
			}) as Me
		}

		reload_me() {
			this.me(this.me_request())
		}

		@$mol_mem
		me(next?: any) {
			if (!next) {
				return this.me_request()
			}

			console.log('set me', next)

			return next ?? null
		}

		me_user_email(): string {
			const me = this.me()
			if (me.user) {
				return me.user.email
			}

			return '???'
		}

		signout() {
			const url = '/api/signout'

			this.$.$mol_fetch.json(url, {
				method: 'post',
				credentials: 'include',
			})

			this.me(null)
		}

		sub() {
			const me = this.me()
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

	type Me = {
		user: {
			id: number
			email: string
			created_at: string
		} | null
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
			// const res = $trip2g_graphql_request_code({
			// 	input: {
			// 		email: this.email(),
			// 	},
			// })

			// if (!res.requestEmailSignInCode) {
			// 	this.request_error('Unknown error')
			// 	return
			// }

			// if (res.message) {

			// }


			// if (res.success) {
			// 	this.$.$mol_state_arg.value('email', this.email())
			// } else if (res.errors) {
			// 	this.request_error(res.errors?.join(', ') ?? 'Unknown error')
			// } else if (res.message) {
			// 	this.request_error(res.message)
			// }
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
			const url = '/api/signinbyemail'

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
				console.log('success', res)
				this.$.$mol_state_arg.value('email', null)
				this.reload_me()
			} else if (res.errors) {
				this.request_error(res.errors?.join(', ') ?? 'Unknown error')
			} else if (res.message) {
				this.request_error(res.message)
			} else {
				alert('Unknown error')
			}
		}
	}
}
