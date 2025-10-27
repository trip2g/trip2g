namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminRestoreBoostyCredentials($input: RestoreBoostyCredentialsInput!) {
			admin {
				payload: restoreBoostyCredentials(input: $input) {
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
	`)

	export class $trip2g_admin_boostycredentials_button_restore extends $.$trip2g_admin_boostycredentials_button_restore {
		restore( event?: Event ) {
			const res = mutate({
				input: {
					id: this.credentials_id()
				}
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'RestoreBoostyCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}