namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
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
	`)

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
			const res = request({
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


	}
}
