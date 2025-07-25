namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_button_restore extends $.$trip2g_admin_patreoncredentials_button_restore {
		@$mol_mem
		restore_enabled() {
			return this.credentials_id() > 0
		}

		restore( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			const res = $trip2g_graphql_request(
				`
					mutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {
						admin {
							restorePatreonCredentials(input: $input) {
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
				`,
				{
					input: {
						id: this.credentials_id()
					}
				}
			)

			if( res.admin.restorePatreonCredentials.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.restorePatreonCredentials.message )
			}

			if( res.admin.restorePatreonCredentials.__typename === 'RestorePatreonCredentialsPayload' ) {
				// Force refresh of data
				this.$.$trip2g_admin_patreoncredentials_catalog.prototype.data( null )
				return
			}

			throw new Error( 'Unexpected response type' )
		}
	}
}