namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminDeleteBoostyCredentials($input: DeleteBoostyCredentialsInput!) {
			admin {
				payload: deleteBoostyCredentials(input: $input) {
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
	`)

	export class $trip2g_admin_boostycredentials_button_delete extends $.$trip2g_admin_boostycredentials_button_delete {
		delete( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			const res = mutate({
				input: {
					id: this.credentials_id()
				}
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'DeleteBoostyCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}