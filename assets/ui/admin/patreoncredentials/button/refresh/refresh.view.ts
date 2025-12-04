namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation RefreshPatreonData($input: RefreshPatreonDataInput!) {
			admin {
				payload: refreshPatreonData(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on RefreshPatreonDataPayload {
						success
					}
				}
			}
		}
	`)

	export class $trip2g_admin_patreoncredentials_button_refresh extends $.$trip2g_admin_patreoncredentials_button_refresh {
		refresh( event?: Event ) {
			const res = mutate( {
				input: {
					credentialsId: this.credentials_id()
				}
			} )

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'RefreshPatreonDataPayload' ) {
				this.status_title( 'Refresh: Success' )
				return
			}

			throw new Error( 'Unexpected response type' )
		}

		@$mol_mem
		override status_title( next?: string ) {
			return next || 'Refresh'
		}
	}
}
