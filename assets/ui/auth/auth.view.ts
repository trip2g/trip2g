namespace $.$$ {
	const signout_mutate = $trip2g_graphql_request(/* GraphQL */ `
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

	const request_email_mutate = $trip2g_graphql_request(/* GraphQL */ `
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
				... on RequestCaptchaPayload {
					__typename
					siteKey
				}
			}
		}
	`)

	const signin_mutate = $trip2g_graphql_request(/* GraphQL */ `
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
	`)

	export class $trip2g_auth extends $.$trip2g_auth {
		me( reset?: null ) {
			return $trip2g_auth_viewer.current( reset )
		}

		reload_me() {
			this.me( null )
		}

		me_user_email(): string {
			return this.me().user?.email || '???'
		}

		signout() {
			const res = signout_mutate()

			if( res.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.data.message )
			}

			if( res.data.__typename === 'SignOutPayload' ) {
				this.me( null )
				return
			}

			throw new Error( 'Unknown error' )
		}

		@$mol_mem
		override entered_email( next?: string ): string {
			// this.$.$mol_state_arg.value( 'email', next || null )

			return next || '' // || this.$.$mol_state_arg.value( 'email' ) || ''
		}

		sub() {
			const viewer = this.me()
			if( viewer.user ) {
				return [ this.AppView() ]
			}

			if( this.entered_email() ) {
				return [ this.CodeForm() ]
			}

			return [ this.EmailForm() ]
		}
	}

	const oauth_error_messages: Record<string, string> = {
		'user_not_found': 'User not registered. Contact administrator.',
		'email_not_verified': 'Email not verified.',
		'oauth_failed': 'Authentication failed. Try again.',
		'invalid_state': 'Invalid request. Try again.',
	}

	export class $trip2g_auth_email_form extends $.$trip2g_auth_email_form {
		@$mol_mem
		request_error( next?: string ): string {
			return next ?? ''
		}

		@$mol_mem
		override email( next?: string ): string {
			const defaultValue = this.$.$trip2g_settings.dev_value( 'hello@example.com' )
			return next || defaultValue
		}

		email_bid() {
			return this.request_error() || ''
		}

		@$mol_mem
		captcha_site_key( next?: string ): string {
			return next ?? ''
		}

		@$mol_mem
		show_captcha( next?: boolean ): boolean {
			return next ?? false
		}

		@$mol_mem
		captcha_token( next?: string | null ): string | null {
			return next ?? null
		}

		static mutate( email: string, captchaToken?: string | null ) {
			const input: { email: string; captchaToken?: string } = { email }
			if( captchaToken ) {
				input.captchaToken = captchaToken
			}
			return request_email_mutate({ input })
		}

		submit() {
			const captchaToken = this.captcha_token()
			const res = this.$.$trip2g_auth_email_form.mutate( this.email(), captchaToken )

			if( res.data.__typename === 'ErrorPayload' ) {
				this.request_error( res.data.message )
				return
			}

			if( res.data.__typename === 'RequestCaptchaPayload' ) {
				this.captcha_site_key( res.data.siteKey )
				this.show_captcha( true )
				return
			}

			if( res.data.__typename === 'RequestEmailSignInCodePayload' ) {
				if( res.data.success ) {
					console.log( 'set email', this.email(), 'code sent' )
					this.entered_email( this.email() )
					this.code_sent( true )
					return
				}
			}

			this.request_error( 'Unknown error' )
		}

		@$mol_mem
		turnstile_loaded(): boolean {
			if( !this.show_captcha() ) return false
			$mol_import.script( 'https://challenges.cloudflare.com/turnstile/v0/api.js' )
			return true
		}

		@$mol_mem
		renderTurnstile() {
			const siteKey = this.captcha_site_key()
			if( !siteKey ) return

			const container = document.getElementById( 'turnstile-container' )
			if( !container ) return

			;( window as any ).turnstile.render( container, {
				sitekey: siteKey,
				callback: ( token: string ) => {
					this.captcha_token( token )
					this.submit()
				},
			} )
		}

		override oauth_error() {
			return this.$.$mol_state_arg.value( 'berror' ) || ''
		}

		override oauth_error_message() {
			const error = this.oauth_error()
			if( !error ) return ''
			return oauth_error_messages[ error ] || error
		}

		override body() {
			const items = [ ...super.body() ]

			// Remove OAuth_error if no error
			const filtered = this.oauth_error()
				? items
				: items.filter( item => item !== this.OAuth_error() )

			// Show captcha container when captcha is required
			if( this.show_captcha() ) {
				this.turnstile_loaded()
				// Schedule renderTurnstile after DOM update
				setTimeout( () => this.renderTurnstile(), 0 )
				return [ ...filtered, this.Captcha_container() ]
			}

			return filtered
		}
	}

	export class $trip2g_auth_code_form extends $.$trip2g_auth_code_form {
		@$mol_mem
		request_error( next?: string ): string {
			return next ?? ''
		}

		@$mol_mem
		override code( next?: string ): string {
			const defaultValue = this.$.$trip2g_settings.dev_value( '111111' )
			return next || defaultValue
		}

		code_bid() {
			return this.request_error() || ''
		}

		submit() {
			const email = this.email()
			if( !email ) {
				this.request_error( 'Email is required' )
				return
			}

			if( !this.code_sent() ) {
				const requestRes = $trip2g_auth_email_form.mutate( email )
				if( requestRes.data.__typename === 'ErrorPayload' ) {
					this.request_error( requestRes.data.message )
					return
				}
			}

			const res = signin_mutate({
				input: {
					email,
					code: this.code(),
				},
			})

			if( res.data.__typename === 'ErrorPayload' ) {
				this.request_error( res.data.message )
				return
			}

			if( res.data.__typename === 'SignInPayload' ) {
				this.$.$mol_state_arg.value( 'email', null )
				this.reload_me()
				return
			}

			this.request_error( 'Unknown error' )
		}
	}
}
