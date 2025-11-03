namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query Admins {
			admin {
				allAdmins {
					nodes {
						id
						grantedAt
						user {
							email
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_admin_catalog extends $.$trip2g_admin_admin_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map( res.admin.allAdmins.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				...this.data().mapKeys( key => this.Content( key ) )
			}
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

		override row_user_email( id: any ): string {
			return this.row( id ).user?.email || '???'
		}

		override row_granted_at( id: any ): string {
			return this.row( id ).grantedAt
		}
	}
}
