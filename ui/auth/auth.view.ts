namespace $.$$ {
	export class $trip2g_auth extends $.$trip2g_auth {
		@$mol_mem
		me(reset?: null) {
			const res = $trip2g_graphql_request(/* GraphQL */ `
				query Viewer {
					viewer {
						id
						user {
							id
							email
							createdAt
						}
					}
				}
			`)

			return res.viewer.user
		}

		reload_me() {
			this.me(null);
		}

		me_user_email(): string {
			return this.me()?.email || '???'
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
							viewer {
								id
							}
						}
					}
				}
			`)

			if (res.data.__typename === 'ErrorPayload') {
				throw new Error(res.data.message)
			}

			if (res.data.__typename === 'SignOutPayload') {
				this.me(null)
				return
			}

			throw new Error('Unknown error')
		}

		sub() {
			const me = this.me()
			if (me) {
				return [this.AppView()]
			}

			const email = this.$.$mol_state_arg.value('email')
			if (email) {
				return [this.CodeForm()]
			}

			const e = this.EmailForm()
			console.log(e);
			return [e];
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

			this.request_error('Unknown error')
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
						code: this.code(),
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

			this.request_error('Unknown error')
		}
	}
}
