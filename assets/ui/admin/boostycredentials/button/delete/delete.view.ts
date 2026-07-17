namespace $.$$ {
	export class $trip2g_admin_boostycredentials_button_delete extends $.$trip2g_admin_boostycredentials_button_delete {
		delete( event?: Event ) {
			event?.preventDefault()
			event?.stopPropagation()

			const res = $trip2g_admin_boostycredentials_button_delete_delete({
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