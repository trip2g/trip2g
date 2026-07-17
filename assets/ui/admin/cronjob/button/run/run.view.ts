namespace $.$$ {
	export class $trip2g_admin_cronjob_button_run extends $.$trip2g_admin_cronjob_button_run {
		run( event?: Event ) {
			const res = $trip2g_admin_cronjob_button_run_save({
				input: {
					id: this.cronjob_id()
				}
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.payload.message )
			}

			if( res.admin.payload.__typename === 'RunCronJobPayload' ) {
				this.status_title( 'Run: Success' )
				return
			}

			throw new Error( 'Unexpected response type' )
		}

		@$mol_mem
		override status_title(next?: string) {
			return next || 'Run'
		}
	}
}