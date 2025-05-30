namespace $.$$ {
	export class $trip2g_admin_release_catalog extends $.$trip2g_admin_release_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminReleases {
					admin {
						allReleases {
							nodes {
								id
								createdAt
								createdBy{
									email
								}
								title
								isLive
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map( res.admin.allReleases.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.AddForm(),
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

		override row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_created_by( id: any ): string {
			const createdBy = this.row( id ).createdBy
			return createdBy?.email || '???'
		}

		override row_title( id: any ): string {
			return this.row( id ).title || '-'
		}

		override row_is_live_status( id: any ): string {
			return this.row( id ).isLive ? 'LIVE' : 'DRAFT'
		}

		override row_is_live_tooltip( id: any ): string {
			return this.row( id ).isLive ? 'This release is currently live' : 'This release is in draft mode'
		}

		override on_create( next?: number ) {
			if( next !== undefined ) {
				this.$.$mol_state_arg.value( 'id', next.toString() )
			}

			return next || 0
		}
	}
}