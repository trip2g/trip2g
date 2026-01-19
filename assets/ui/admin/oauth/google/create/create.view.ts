namespace $.$$ {
	const urls_query = $trip2g_graphql_request(/* GraphQL */ `
		query GoogleOAuthUrls($input: OAuthUrlInput!) {
			publicUrl
			googleAuthUrl(input: $input) {
				callbackUrl
			}
		}
	`)

	const create_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreateGoogleOAuthCredentials($input: CreateGoogleOAuthCredentialsInput!) {
			admin {
				data: createGoogleOAuthCredentials(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on CreateGoogleOAuthCredentialsPayload {
						credentials {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_google_create extends $.$trip2g_admin_oauth_google_create {
		@$mol_mem
		urls() {
			return urls_query({ input: { redirectUrl: '/', dry: true } })
		}

		override homepage_url() {
			return this.urls().publicUrl
		}

		override callback_url() {
			return this.urls().googleAuthUrl.callbackUrl
		}

		@$mol_mem
		override result( next?: string ) {
			return next ?? ''
		}

		submit() {
			this.result( '' )

			const res = create_mutation({
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
