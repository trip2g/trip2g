namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminChangeWebhooks {
			admin {
				allChangeWebhooks {
					nodes {
						id
						url
						enabled
						description
						onCreate
						onUpdate
						onRemove
						lastDeliveryAt
						lastDeliveryStatus
						createdAt
					}
				}
			}
		}
	`)

	export class $trip2g_admin_changewebhook_catalog extends $.$trip2g_admin_changewebhook_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
			return $trip2g_graphql_make_map( res.admin.allChangeWebhooks.nodes )
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

		override row_enabled( id: any ): string {
			return this.row( id ).enabled ? 'Yes' : 'No'
		}

		override row_events( id: any ): string {
			const r = this.row( id )
			const events: string[] = []
			if( r.onCreate ) events.push( 'create' )
			if( r.onUpdate ) events.push( 'update' )
			if( r.onRemove ) events.push( 'remove' )
			return events.join( ', ' ) || 'none'
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
