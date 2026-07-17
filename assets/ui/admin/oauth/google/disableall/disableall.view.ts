namespace $.$$ {
	export class $trip2g_admin_oauth_google_disableall extends $.$trip2g_admin_oauth_google_disableall {
		disable() {
			const res = $trip2g_admin_oauth_google_disableall_confirm()

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			this.$.$mol_state_arg.value( 'id', null )
		}
	}
}
