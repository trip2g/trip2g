namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_button_restore extends $.$trip2g_admin_patreoncredentials_button_restore {
		restore( event?: Event ) {
			const res = $trip2g_graphql_request(
				`
					mutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {
						admin {
							restorePatreonCredentials(input: $input) {
								... on ErrorPayload {
									message
								}
								... on RestorePatreonCredentialsPayload {
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
						id: this.credentials_id()
					}
				}
			)

			if( res.admin.restorePatreonCredentials.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.restorePatreonCredentials.message )
			}

			if( res.admin.restorePatreonCredentials.__typename === 'RestorePatreonCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}