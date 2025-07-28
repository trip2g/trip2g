namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_button_delete extends $.$trip2g_admin_patreoncredentials_button_delete {
		delete( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			const res = $trip2g_graphql_request(
				`
					mutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {
						admin {
							deletePatreonCredentials(input: $input) {
								... on ErrorPayload{
									message
								}
								... on DeletePatreonCredentialsPayload {
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

			if( res.admin.deletePatreonCredentials.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.deletePatreonCredentials.message )
			}

			if( res.admin.deletePatreonCredentials.__typename === 'DeletePatreonCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}