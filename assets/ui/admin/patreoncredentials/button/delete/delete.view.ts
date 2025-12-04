namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {
			admin {
				payload: deletePatreonCredentials(input: $input) {
					__typename
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
	`)

	export class $trip2g_admin_patreoncredentials_button_delete extends $.$trip2g_admin_patreoncredentials_button_delete {
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

			if( res.admin.payload.__typename === 'DeletePatreonCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}
