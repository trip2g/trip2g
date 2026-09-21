namespace $.$$ {
	export class $trip2g_auth_oauth_buttons extends $.$trip2g_auth_oauth_buttons {
		@$mol_mem
		auth_methods() {
			return $trip2g_auth_methods_here()
		}

		override google_uri() {
			return this.auth_methods()?.googleAuthUrl.authUrl || ''
		}

		override github_uri() {
			return this.auth_methods()?.githubAuthUrl.authUrl || ''
		}

		override oidc_uri() {
			return this.auth_methods()?.oidcAuthUrl.authUrl || ''
		}

		override oidc_title() {
			return this.auth_methods()?.oidcAuthUrl.label || super.oidc_title()
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
