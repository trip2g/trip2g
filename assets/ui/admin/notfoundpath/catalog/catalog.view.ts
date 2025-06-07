namespace $.$$ {
	export class $trip2g_admin_notfoundpath_catalog extends $.$trip2g_admin_notfoundpath_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminNotFoundPaths {
					admin {
						allNotFoundPaths {
							nodes {
								id
								path
								totalHits
								lastHitAt
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map( res.admin.allNotFoundPaths.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids()
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

		override row_path( id: any ): string {
			return this.row( id ).path
		}

		override row_total_hits( id: any ): string {
			return this.row( id ).totalHits.toString()
		}

		override row_last_hit_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).lastHitAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}
	}
}