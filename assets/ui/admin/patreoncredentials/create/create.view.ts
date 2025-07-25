namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_create extends $.$trip2g_admin_patreoncredentials_create {
		override body() {
			if( this.credentials_id_string() !== '' ) {
				return [ this.CredentialsView() ]
			}

			return super.body()
		}

		override token_bid(): string {
			const token = this.token()
			if( !token.trim() ) {
				return 'Creator Access Token is required'
			}

			if( token.length < 10 ) {
				return 'Token must be at least 10 characters'
			}

			return ''
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreatePatreonCreds($input: CreatePatreonCredentialsInput!) {
						admin {
							createPatreonCredentials(input: $input) {
								... on ErrorPayload {
									message
								}
								... on CreatePatreonCredentialsPayload {
									patreonCredentials {
										id
									}
								}
							}
						}
					}
				`,
				{
					input: {
						creatorAccessToken: this.token()
					},
				}
			)

			if( res.admin.createPatreonCredentials.__typename === 'ErrorPayload' ) {
				this.result( res.admin.createPatreonCredentials.message )
				return
			}

			if( res.admin.createPatreonCredentials.__typename === 'CreatePatreonCredentialsPayload' ) {
				this.credentials_id_string( res.admin.createPatreonCredentials.patreonCredentials.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}