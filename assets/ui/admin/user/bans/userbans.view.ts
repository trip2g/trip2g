namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminListUserBans {
			admin {
				allUserUserBans {
					nodes {
						id: userId
						user {
							__typename
							email
						}
						bannedBy {
							user {
								email
							}
						}
						createdAt
						reason
					}
				}
			}
		}
	`)
	export class $trip2g_admin_user_bans extends $.$trip2g_admin_user_bans {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map( res.admin.allUserUserBans.nodes )
		}

		body() {
			return this.data().map( key => this.Row( key ), this.NoRows() )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_user_id( id: any ): number {
			return id
		}

		row_user_email( id: any ): string {
			return this.row( id ).user.email || '-'
		}

		row_banned_by_email( id: any ): string {
			return this.row( id ).bannedBy?.user.email || '-'
		}

		row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		row_reason( id: any ): string {
			return this.row( id ).reason
		}
	}
}
