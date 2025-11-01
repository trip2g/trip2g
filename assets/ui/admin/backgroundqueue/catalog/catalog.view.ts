namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminBackgroundQueues {
			admin {
				allBackgroundQueues {
					nodes {
						id
						pendingCount
						retryCount
						stopped
					}
				}
			}
		}
	`)

	export class $trip2g_admin_backgroundqueue_catalog extends $.$trip2g_admin_backgroundqueue_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request({})

			return $trip2g_graphql_make_map( res.admin.allBackgroundQueues.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: null, // No add form for this catalog
				...this.data().mapKeys( key => this.Content( key ) )
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): string {
			return id
		}

		override row_name( id: any ): string {
			return this.row( id ).id
		}

		override row_pending( id: any ): string {
			return this.row( id ).pendingCount.toString()
		}

		override row_retry( id: any ): string {
			return this.row( id ).retryCount.toString()
		}

		override row_status( id: any ): string {
			return this.row( id ).stopped ? 'Stopped' : 'Running'
		}
	}
}
