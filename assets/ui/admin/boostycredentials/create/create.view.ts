namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateBoostyCreds($input: CreateBoostyCredentialsInput!) {
			admin {
				payload: createBoostyCredentials(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on CreateBoostyCredentialsPayload {
						boostyCredentials {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_boostycredentials_create extends $.$trip2g_admin_boostycredentials_create {
		override body() {
			if( this.credentials_id_string() !== '' ) {
				return [ this.CredentialsView() ]
			}

			return super.body()
		}

		override auth_data_bid(): string {
			const authData = this.auth_data()
			if( !authData.trim() ) {
				return 'Auth Data is required'
			}

			if( authData.length < 10 ) {
				return 'Auth Data must be at least 10 characters'
			}

			return ''
		}

		override device_id_bid(): string {
			const deviceId = this.device_id()
			if( !deviceId.trim() ) {
				return 'Device ID is required'
			}

			return ''
		}

		override blog_name_bid(): string {
			const blogName = this.blog_name()
			if( !blogName.trim() ) {
				return 'Blog Name is required'
			}

			return ''
		}

		submit() {
			const res = mutate({
				input: {
					authData: this.auth_data(),
					deviceId: this.device_id(),
					blogName: this.blog_name()
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateBoostyCredentialsPayload' ) {
				this.credentials_id_string( res.admin.payload.boostyCredentials.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}