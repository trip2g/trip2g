namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {
			admin {
				payload: restorePatreonCredentials(input: $input) {
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
	`)
	export class $trip2g_admin_patreoncredentials_button_restore extends $.$trip2g_admin_patreoncredentials_button_restore {
		restore( event?: Event ) {
			const res = mutate({
				input: {
					id: this.credentials_id()
				}
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'RestorePatreonCredentialsPayload' ) {
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}