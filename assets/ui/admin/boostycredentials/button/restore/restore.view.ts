namespace $.$$ {
	export class $trip2g_admin_boostycredentials_button_restore extends $.$trip2g_admin_boostycredentials_button_restore {
		restore( event?: Event ) {
			const res = $trip2g_graphql_request(
				`
					mutation AdminRestoreBoostyCredentials($input: RestoreBoostyCredentialsInput!) {
						admin {
							restoreBoostyCredentials(input: $input) {
								... on ErrorPayload {
									message
								}
								... on RestoreBoostyCredentialsPayload {
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
						id: this.credentials_id()
					}
				}
			)

			if( res.admin.restoreBoostyCredentials.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.restoreBoostyCredentials.message )
			}

			if( res.admin.restoreBoostyCredentials.__typename === 'RestoreBoostyCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}