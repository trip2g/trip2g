namespace $.$$ {
	export class $trip2g_admin_apikey_catalog extends $.$trip2g_admin_apikey_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminListApiKeys {
					admin {
						allApiKeys {
							nodes {
								id
								createdAt
								description
								createdBy {
									id
									email
								}
								disabledAt
								disabledBy {
									id
									email
								}
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map( res.admin.allApiKeys.nodes )
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

		override row_description( id: any ): string {
			return this.row( id ).description || '-'
		}

		override row_disabled_at( id: any ): string {
			const v = this.row(id).disabledAt
			if (!v) {
				return '-'
			}

			const m = new $mol_time_moment( v )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_disabled_by( id: any ): string {
			return this.row( id ).disabledBy?.email || ''
		}

		override row_disabled(id: any): boolean {
			return !!this.row( id ).disabledAt
		}
	}
}