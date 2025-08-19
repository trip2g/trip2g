namespace $.$$ {
	export class $trip2g_admin_cronjob_show extends $.$trip2g_admin_cronjob_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return super.body()
		}

		@$mol_mem
		cronjob_data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminCronJobShow($id: Int64!) {
					admin {
						cronJob(id: $id) {
							id
							name
							enabled
							expression
							lastExecAt
						}
					}
				}
			`, {
				id: this.cronjob_id()
			})

			return res.admin.cronJob
		}

		override title(): string {
			const data = this.cronjob_data()
			return data ? `Cron Job: ${data.name}` : 'Cron Job'
		}

		cronjob_id_string(): string {
			const data = this.cronjob_data()
			return data ? data.id.toString() : '-'
		}

		cronjob_name_value(): string {
			const data = this.cronjob_data()
			return data ? data.name : '-'
		}

		cronjob_expression(): string {
			const data = this.cronjob_data()
			return data ? data.expression : '-'
		}

		cronjob_enabled_status(): string {
			const data = this.cronjob_data()
			return data ? (data.enabled ? 'Enabled' : 'Disabled') : '-'
		}

		cronjob_last_exec_at(): string {
			const data = this.cronjob_data()
			if (!data || !data.lastExecAt) return 'Never'
			const m = new $mol_time_moment(data.lastExecAt)
			return m.toString('YYYY-MM-DD HH:mm:ss')
		}

		@$mol_mem
		run_click( next?: any ): any {
			if ( next === undefined ) return null

			const res = $trip2g_graphql_request( `
				mutation AdminRunCronJob($input: RunCronJobInput!) {
					admin {
						runCronJob(input: $input) {
							... on RunCronJobPayload {
								execution {
									id
								}
							}
							... on ErrorPayload {
								message
							}
						}
					}
				}
			`, {
				input: {
					id: this.cronjob_id()
				}
			})

			const result = res.admin.runCronJob
			if (result.__typename === 'ErrorPayload') {
				throw new Error(result.message)
			}

			this.Executions().data(null)
			
			return null
		}

	}
}
