namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_button_refresh extends $.$trip2g_admin_patreoncredentials_button_refresh {
		refresh( event?: Event ) {
			const res = $trip2g_graphql_request(
				`
					mutation RefreshPatreonData($input: RefreshPatreonDataInput!) {
						admin {
							refreshPatreonData(input: $input) {
								... on ErrorPayload {
									message
								}
								... on RefreshPatreonDataPayload {
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

			if( res.admin.refreshPatreonData.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.refreshPatreonData.message )
			}

			if( res.admin.refreshPatreonData.__typename === 'RefreshPatreonDataPayload' ) {
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