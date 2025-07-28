namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_button_delete extends $.$trip2g_admin_patreoncredentials_button_delete {
		delete( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			if (!confirm('Are you sure you want to delete these Patreon credentials?')) {
				return
			}

			const res = $trip2g_graphql_request(
				`
					mutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {
						admin {
							deletePatreonCredentials(input: $input) {
								... on ErrorPayload{
									message
								}
								... on DeletePatreonCredentialsPayload {
									deletedId
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