namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminRunCronJob($input: RunCronJobInput!) {
			admin {
				runCronJob(input: $input) {
					__typename
					... on RunCronJobPayload {
						execution {
							id
							job {
								id
							}
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)
	export class $trip2g_admin_cronjob_button_run extends $.$trip2g_admin_cronjob_button_run {
		run( event?: Event ) {
			const res = mutate({
				input: {
					id: this.cronjob_id()
				}
			})

			if( res.admin.runCronJob.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.runCronJob.message )
			}

			if( res.admin.runCronJob.__typename === 'RunCronJobPayload' ) {
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