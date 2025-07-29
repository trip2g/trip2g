namespace $.$$ {
	export class $trip2g_admin_boostycredentials_button_delete extends $.$trip2g_admin_boostycredentials_button_delete {
		delete( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			const res = $trip2g_graphql_request(
				`
					mutation AdminDeleteBoostyCredentials($input: DeleteBoostyCredentialsInput!) {
						admin {
							deleteBoostyCredentials(input: $input) {
								... on ErrorPayload{
									message
								}
								... on DeleteBoostyCredentialsPayload {
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

			if( res.admin.deleteBoostyCredentials.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.deleteBoostyCredentials.message )
			}

			if( res.admin.deleteBoostyCredentials.__typename === 'DeleteBoostyCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}