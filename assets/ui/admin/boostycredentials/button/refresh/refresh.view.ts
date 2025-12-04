namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation RefreshBoostyData($input: RefreshBoostyDataInput!) {
			admin {
				payload: refreshBoostyData(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on RefreshBoostyDataPayload {
						success
						credentials {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_boostycredentials_button_refresh extends $.$trip2g_admin_boostycredentials_button_refresh {
		refresh( event?: Event ) {
			const res = mutate({
				input: {
					credentialsId: this.credentials_id()
				}
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'RefreshBoostyDataPayload' ) {
				this.status_title( 'Refresh: Success' )
				return
			}

			throw new Error( 'Unexpected response type' )
		}

		@$mol_mem
		override status_title(next?: string) {
			return next || 'Refresh'
		}
	}
}