namespace $.$$ {
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
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateBoostyCreds($input: CreateBoostyCredentialsInput!) {
						admin {
							createBoostyCredentials(input: $input) {
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
				`,
				{
					input: {
						authData: this.auth_data(),
						deviceId: this.device_id(),
						blogName: this.blog_name()
					},
				}
			)

			if( res.admin.createBoostyCredentials.__typename === 'ErrorPayload' ) {
				this.result( res.admin.createBoostyCredentials.message )
				return
			}

			if( res.admin.createBoostyCredentials.__typename === 'CreateBoostyCredentialsPayload' ) {
				this.credentials_id_string( res.admin.createBoostyCredentials.boostyCredentials.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}