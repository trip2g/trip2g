namespace $.$$ {
	export class $trip2g_admin_purchase_catalog extends $.$trip2g_admin_purchase_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminPurchases {
					admin {
						allPurchases {
							nodes {
								id
								createdAt
								paymentProvider
								status
								successful
								offerId
								email
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map( res.admin.allPurchases.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): string {
			return this.row( id ).id
		}

		override row_email( id: any ): string {
			return this.row( id ).email || '-'
		}

		override row_payment_provider( id: any ): string {
			return this.row( id ).paymentProvider || '-'
		}

		override row_status( id: any ): string {
			return this.row( id ).status || '-'
		}

		override row_successful( id: any ): string {
			return this.row( id ).successful ? 'Yes' : 'No'
		}

		override row_offer_id( id: any ): string {
			return this.row( id ).offerId?.toString() || '-'
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}
	}
}