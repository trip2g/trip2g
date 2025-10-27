namespace $.$$ {
	const request = $trip2g_graphql_request( /* GraphQL */ `
		query AdminTgBots {
			admin {
				allTgBots {
					nodes {
						id
						name
						description
						enabled
						createdAt
						createdBy {
							email
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_tgbot_catalog extends $.$trip2g_admin_tgbot_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
			return $trip2g_graphql_make_map( res.admin.allTgBots.nodes )
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

		override row_name( id: any ): string {
			return this.row( id ).name || '-'
		}

		override row_description( id: any ): string {
			return this.row( id ).description || '-'
		}

		override row_enabled_status( id: any ): string {
			return this.row( id ).enabled ? 'Enabled' : 'Disabled'
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_created_by( id: any ): string {
			const createdBy = this.row( id ).createdBy
			return createdBy?.email || '???'
		}
	}
}
