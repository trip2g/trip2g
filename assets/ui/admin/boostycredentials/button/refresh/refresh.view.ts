namespace $.$$ {
	export class $trip2g_admin_boostycredentials_button_refresh extends $.$trip2g_admin_boostycredentials_button_refresh {
		refresh( event?: Event ) {
			const res = $trip2g_graphql_request(
				`
					mutation RefreshBoostyData($input: RefreshBoostyDataInput!) {
						admin {
							refreshBoostyData(input: $input) {
								... on ErrorPayload {
									message
								}
								... on RefreshBoostyDataPayload {
									success
								}
							}
						}
					}
				`,
				{
					input: {
						credentialsId: this.credentials_id()
					}
				}
			)

			if( res.admin.refreshBoostyData.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.refreshBoostyData.message )
			}

			if( res.admin.refreshBoostyData.__typename === 'RefreshBoostyDataPayload' ) {
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