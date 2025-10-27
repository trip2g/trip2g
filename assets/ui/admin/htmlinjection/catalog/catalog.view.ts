namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminHtmlInjections {
				admin {
					allHtmlInjections {
						nodes {
							id
							createdAt
							activeFrom
							activeTo
							description
							position
							placement
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_htmlinjection_catalog extends $.$trip2g_admin_htmlinjection_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request()

			return $trip2g_graphql_make_map( res.admin.allHtmlInjections.nodes )
		}

		override after_delete( id: any ) {
			this.spread( '' )
			this.data( null )
		}

		override after_create( id?: number ) {
			this.spread( `key${ id }` )
			this.data( null )
			return id || 0
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.CreateForm(),
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' && !id.startsWith( 'update/' ) )
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

		override row_description( id: any ): string {
			return this.row( id ).description || '-'
		}

		override row_placement( id: any ): string {
			return this.row( id ).placement
		}

		override row_position( id: any ): string {
			return this.row( id ).position.toString()
		}

		override row_active_from( id: any ): string {
			const activeFrom = this.row( id ).activeFrom
			if( !activeFrom ) return '-'
			const m = new $mol_time_moment( activeFrom )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_active_to( id: any ): string {
			const activeTo = this.row( id ).activeTo
			if( !activeTo ) return '-'
			const m = new $mol_time_moment( activeTo )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}
	}
}
