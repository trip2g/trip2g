namespace $.$$ {
	const oauth_urls_query = $trip2g_graphql_request(/* GraphQL */ `
		query OAuthUrls($input: OAuthUrlInput!) {
			googleAuthUrl(input: $input) {
				authUrl
			}
			githubAuthUrl(input: $input) {
				authUrl
			}
		}
	`)

	export class $trip2g_auth_oauth_buttons extends $.$trip2g_auth_oauth_buttons {
		@$mol_mem
		oauth_urls() {
			const redirectUrl = this.$.$mol_state_arg.href()
			try {
				return oauth_urls_query({ input: { redirectUrl } })
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

		override buttons() {
			const list: $mol_view[] = []
			if( this.google_uri() ) {
				list.push( this.Google() )
			}
			if( this.github_uri() ) {
				list.push( this.Github() )
			}
			return list
		}
	}
}
