namespace $.$$ {
	export class $trip2g_auth_oauth_buttons extends $.$trip2g_auth_oauth_buttons {
		@$mol_mem
		oauth_urls() {
			const redirectUrl = this.$.$mol_state_arg.href()
			try {
				return $trip2g_auth_oauth_buttons_urls({ input: { redirectUrl } })
			} catch {
				return null
			}
		}

		override google_uri() {
			return this.oauth_urls()?.googleAuthUrl.authUrl || ''
		}

		override github_uri() {
			return this.oauth_urls()?.githubAuthUrl.authUrl || ''
		}

		override oidc_uri() {
			return this.oauth_urls()?.oidcAuthUrl.authUrl || ''
		}

		override buttons() {
			const list: $mol_view[] = []
			if( this.google_uri() ) {
				list.push( this.Google() )
			}
			if( this.github_uri() ) {
				list.push( this.Github() )
			}
			if( this.oidc_uri() ) {
				list.push( this.Oidc() )
			}
			return list
		}
	}
}
