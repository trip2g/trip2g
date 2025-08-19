namespace $.$$ {
	export class $trip2g_admin_cronjob_show_executions extends $.$trip2g_admin_cronjob_show_executions {
		@$mol_mem
		override data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminCronJobExecutions($id: Int64!) {
					admin {
						cronJob(id: $id) {
							id
							executions {
								id
								startedAt
								finishedAt
								status
								errorMessage
							}
						}
					}
				}
			`, {
				id: this.cronjob_id()
			} )

			if (!res.admin.cronJob || !res.admin.cronJob.executions) {
				return $trip2g_graphql_make_map( [] )
			}

			return $trip2g_graphql_make_map( res.admin.cronJob.executions )
		}

		@$mol_mem
		override data_rows() {
			return this.data().map( id => this.Row( id ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_started_at( id: any ): string {
			const startedAt = this.row( id ).startedAt
			if (!startedAt) return 'Unknown'
			const m = new $mol_time_moment( startedAt )
			return m.toString( 'YYYY-MM-DD HH:mm:ss' )
		}

		override row_finished_at( id: any ): string {
			const finishedAt = this.row( id ).finishedAt
			if (!finishedAt) return 'Running...'
			const m = new $mol_time_moment( finishedAt )
			return m.toString( 'YYYY-MM-DD HH:mm:ss' )
		}

		override row_status( id: any ): string {
			return this.row( id ).status || 'Unknown'
		}

		override row_error_message( id: any ): string {
			return this.row( id ).errorMessage || '-'
		}
	}
}
