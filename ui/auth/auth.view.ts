namespace $.$$ {
	export class $trip2g_auth extends $.$trip2g_auth {
		me(reset?: null) {
			return $trip2g_auth_viewer.current(reset)
		}

		reload_me() {
			this.me(null);
		}

		me_user_email(): string {
			return this.me().user?.email || '???'
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

		@$mol_mem
		override entered_email(next?: string): string {
			this.$.$mol_state_arg.value('email', next || null)

			return next || this.$.$mol_state_arg.value('email') || ''
		}

		sub() {
			console.log('entered_email', this.entered_email())

			const viewer = this.me()
			if (viewer.user) {
				return [this.AppView()]
			}

			if (this.entered_email()) {
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

		static mutate(email: string) {
			return $trip2g_graphql_request(
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
					input: { email },
				}
			)
		}

		submit() {
			const res = this.$.$trip2g_auth_email_form.mutate(this.email())

			if (res.data.__typename === 'ErrorPayload') {
				this.request_error(res.data.message)
				return
			}

			if (res.data.__typename === 'RequestEmailSignInCodePayload') {
				if (res.data.success) {
					console.log('set email', this.email(), 'code sent')
					this.entered_email(this.email())
					this.code_sent(true)
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

		submit() {
			const email = this.email()
			if (!email) {
				this.request_error('Email is required')
				return
			}

			if (!this.code_sent()) {
				const requestRes = $trip2g_auth_email_form.mutate(email);
				if (requestRes.data.__typename === 'ErrorPayload') {
					this.request_error(requestRes.data.message)
					return
				}
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
