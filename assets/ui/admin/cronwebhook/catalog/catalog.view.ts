namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminCronWebhooks {
			admin {
				allCronWebhooks {
					nodes {
						id
						url
						cronSchedule
						enabled
						description
						nextRunAt
						lastDeliveryAt
						lastDeliveryStatus
						createdAt
					}
				}
			}
		}
	`)

	export class $trip2g_admin_cronwebhook_catalog extends $.$trip2g_admin_cronwebhook_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
			return $trip2g_graphql_make_map( res.admin.allCronWebhooks.nodes )
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

		override row_url( id: any ): string {
			return this.row( id ).url
		}

		override row_schedule( id: any ): string {
			return this.row( id ).cronSchedule
		}

		override row_enabled( id: any ): string {
			return this.row( id ).enabled ? 'Yes' : 'No'
		}

		override row_next_run( id: any ): string {
			const r = this.row( id )
			if( !r.nextRunAt ) return '-'
			const m = new $mol_time_moment( r.nextRunAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		override row_last_delivery( id: any ): string {
			const r = this.row( id )
			if( !r.lastDeliveryAt ) return '-'
			const m = new $mol_time_moment( r.lastDeliveryAt )
			const status = r.lastDeliveryStatus || ''
			return m.toString( 'YYYY-MM-DD HH:mm' ) + ( status ? ` (${status})` : '' )
		}
	}
}
