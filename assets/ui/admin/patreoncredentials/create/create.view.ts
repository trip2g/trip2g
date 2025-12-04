namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreatePatreonCreds($input: CreatePatreonCredentialsInput!) {
			admin {
				payload: createPatreonCredentials(input: $input) {
					__typename
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
	`)

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
			const res = mutate({
				input: {
					creatorAccessToken: this.token()
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreatePatreonCredentialsPayload' ) {
				this.credentials_id_string( res.admin.payload.patreonCredentials.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}