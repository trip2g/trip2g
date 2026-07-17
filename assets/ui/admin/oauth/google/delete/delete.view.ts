namespace $.$$ {
	export class $trip2g_admin_oauth_google_delete extends $.$trip2g_admin_oauth_google_delete {
		delete() {
			const res = $trip2g_admin_oauth_google_delete_confirm({
				input: { id: this.credentials_id() },
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			if( res.admin.data.__typename === 'DeleteGoogleOAuthCredentialsPayload' ) {
				this.$.$mol_state_arg.value( 'id', null )
				this.$.$mol_state_arg.value( 'action', null )
			}
		}
	}
}
