namespace $.$$ {
	const state = $trip2g_graphql_boosty_credentials_state_enum

	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminBoostyCredentials($filter: AdminBoostyCredentialsFilterInput) {
			admin {
				allBoostyCredentials(filter: $filter) {
					nodes {
						id
						state
						deviceId
						blogName
						createdAt
						createdBy {
							id
							email
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_boostycredentials_catalog extends $.$trip2g_admin_boostycredentials_catalog {
		@$mol_mem
		filter_state() {
			const filter = this.$.$mol_state_arg.value( 'filter' ) || 'all'
			switch( filter ) {
				case 'active':
					return state.Active
				case 'deleted':
					return state.Deleted
				default:
					return null
			}
		}

		@$mol_mem
		data( reset?: null ) {
			const filter = this.filter_state()
			const res = request({
				filter: filter ? { state: filter } : null
			})

			return $trip2g_graphql_make_map( res.admin.allBoostyCredentials.nodes )
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

		override row_state( id: any ): string {
			const state = this.row( id ).state
			return state === 'ACTIVE' ? 'Active' : 'Deleted'
		}

		override row_blog_name( id: any ): string {
			return this.row( id ).blogName
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}
	}
}