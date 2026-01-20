namespace $.$$ {
	const list_query = $trip2g_graphql_request(/* GraphQL */ `
		query AdminGoogleOAuthCredentials {
			admin {
				allGoogleOAuthCredentials {
					nodes {
						id
						name
						clientId
						active
						createdAt
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_google_catalog extends $.$trip2g_admin_oauth_google_catalog {
		@$mol_mem
		data( reset?: null ) {
			return $trip2g_graphql_make_map( list_query().admin.allGoogleOAuthCredentials.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.AddForm(),
				disableall: this.DisableallForm(),
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		@$mol_mem
		override menu_body() {
			const body = super.menu_body()
			if (this.data().size() === 0) {
				return [this.Empty()]
			}

			return body
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' && id !== 'disableall' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_id( id: any ): number {
			return this.row( id ).id
		}

		row_id_string( id: any ) {
			return this.row( id ).id.toString()
		}

		row_name( id: any ) {
			return this.row( id ).name
		}

		row_client_id( id: any ) {
			const clientId = this.row( id ).clientId
			if( clientId.length > 24 ) {
				return clientId.slice( 0, 10 ) + '...' + clientId.slice( -10 )
			}
			return clientId
		}

		row_created_at( id: any ) {
			return this.row( id ).createdAt
		}
	}
}
