namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminAllCronJobs {
			admin {
				allCronJobs {
					nodes {
						id
						name
						enabled
						expression
						lastExecAt
					}
				}
			}
		}
	`)
	export class $trip2g_admin_cronjob_catalog extends $.$trip2g_admin_cronjob_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map( res.admin.allCronJobs.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => !id.startsWith( 'update/' ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_name( id: any ): string {
			return this.row( id ).name || '-'
		}

		override row_enabled_status( id: any ): string {
			return this.row( id ).enabled ? 'Enabled' : 'Disabled'
		}

		override row_expression( id: any ): string {
			return this.row( id ).expression || '-'
		}

		override row_last_exec_at( id: any ): string {
			const lastExecAt = this.row( id ).lastExecAt
			if (!lastExecAt) return 'Never'
			const m = new $mol_time_moment( lastExecAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}
	}
}