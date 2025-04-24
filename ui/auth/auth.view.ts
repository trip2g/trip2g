namespace $.$$ {
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
			const res = $trip2g_graphql_request(/* GraphQL */ `
				mutation SignOut {
					data: signOut {
						... on ErrorPayload {
							__typename
							message
						}
						... on SignOutPayload {
							__typename
							viewer { id }
						}
					}
				}
			`)

			if (res.data.__typename === 'ErrorPayload') {
				throw new Error(res.data.message)
			}

			if (res.data.__typename === 'SignOutPayload') {
				this.me(null);
				return
			}
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
			const res = $trip2g_graphql_request(
				/* GraphQL */ `
					mutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {
						data: requestEmailSignInCode(input: $input) {
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
				`,
				{
					input: {
						email: this.email(),
					},
				}
			)

			if (res.data.__typename === 'ErrorPayload') {
				this.request_error(res.data.message)
				return
			}

			if (res.data.__typename === 'RequestEmailSignInCodePayload') {
				if (res.data.success) {
					this.$.$mol_state_arg.value('email', this.email())
					return
				}
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
			const email = this.email()
			if (!email) {
				this.request_error('Email is required')
				return
			}

			const res = $trip2g_graphql_request(
				/* GraphQL */ `
					mutation SignInByEmail($input: SignInByEmailInput!) {
						data: signInByEmail(input: $input) {
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
				`,
				{
					input: {
						email,
						code: parseInt(this.code(), 10),
					},
				}
			)

			if (res.data.__typename === 'ErrorPayload') {
				this.request_error(res.data.message)
				return
			}

			if (res.data.__typename === 'SignInPayload') {
				this.$.$mol_state_arg.value('email', null)
				this.reload_me()
				return
			}
		}
	}
}
