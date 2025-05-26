namespace $.$$ {
	export class $trip2g_admin_list_apikeys extends $.$trip2g_admin_list_apikeys {
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
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map( res.admin.allApiKeys.nodes )
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys( key => this.Content( key ) )
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
			const m = new $mol_time_moment(this.row(id).createdAt)
			return m.toString('YYYY-MM-DD')
		}

		override row_created_by( id: any ): string {
			const createdBy = this.row( id ).createdBy
			return createdBy?.email || '???'
		}

		override row_description( id: any ): string {
			return this.row( id ).description || '-'
		}
	}
}