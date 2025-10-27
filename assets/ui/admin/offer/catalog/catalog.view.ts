namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminOffers {
			admin {
				allOffers {
					nodes {
						id
						publicId
						createdAt
						lifetime
						priceUSD
						startsAt
						endsAt
						subgraphs {
							name
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_offer_catalog extends $.$trip2g_admin_offer_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map( res.admin.allOffers.nodes )
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

		override row_subgraphs( id: any ): string {
			return this.row( id ).subgraphs.map(s => s.name).join( ', ' )
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_public_id( id: any ): string {
			return this.row( id ).publicId || '-'
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_price_usd( id: any ): string {
			const price = this.row( id ).priceUSD
			return price ? `$${ price.toFixed( 2 ) }` : '-'
		}

		override row_lifetime( id: any ): string {
			return this.row( id ).lifetime || '-'
		}

		override row_starts_at( id: any ): string {
			const startsAt = this.row( id ).startsAt
			if( !startsAt ) return '-'
			const m = new $mol_time_moment( startsAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_ends_at( id: any ): string {
			const endsAt = this.row( id ).endsAt
			if( !endsAt ) return '-'
			const m = new $mol_time_moment( endsAt )
			return m.toString( 'YYYY-MM-DD' )
		}
	}
}