namespace $.$$ {
	const request = $trip2g_graphql_request( /* GraphQL */ `
		query AdminTelegramAccounts {
			admin {
				allTelegramAccounts {
					nodes {
						id
						phone
						displayName
						isPremium
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

	export class $trip2g_admin_telegramaccount_catalog extends $.$trip2g_admin_telegramaccount_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
			return $trip2g_graphql_make_map( res.admin.allTelegramAccounts.nodes )
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

		override row_phone( id: any ): string {
			return this.row( id ).phone || '-'
		}

		override row_display_name( id: any ): string {
			return this.row( id ).displayName || '-'
		}

		override row_enabled_status( id: any ): string {
			return this.row( id ).enabled ? 'Enabled' : 'Disabled'
		}

		override row_premium_status( id: any ): string {
			return this.row( id ).isPremium ? 'Yes' : 'No'
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
