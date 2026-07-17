namespace $.$$ {
	export class $trip2g_admin_oauth_github_create extends $.$trip2g_admin_oauth_github_create {
		@$mol_mem
		urls() {
			return $trip2g_admin_oauth_github_create_urls({ input: { redirectUrl: '/', dry: true } })
		}

		override homepage_url() {
			return this.urls().publicUrl
		}

		override callback_url() {
			return this.urls().githubAuthUrl.callbackUrl
		}

		@$mol_mem
		override result( next?: string ) {
			return next ?? ''
		}

		submit() {
			this.result( '' )

			const res = $trip2g_admin_oauth_github_create_submit({
				input: {
					name: this.name(),
					clientId: this.client_id(),
					clientSecret: this.client_secret(),
				},
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			this.result( 'Created!' )
			this.name( '' )
			this.client_id( '' )
			this.client_secret( '' )
		}
	}
}
