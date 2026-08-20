namespace $.$$ {
	export class $trip2g_admin_deliverytrace_catalog extends $.$trip2g_admin_deliverytrace_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_deliverytrace_catalog_list( { withEmpty: this.with_empty() } )
			// Keyed by trace id: it is already the chain's identity, so the menu
			// key, the URL arg and the detail page's query are all the same string.
			return new Map( res.admin.deliveryTraces.map( row => [ row.trace ?? '', row ] as const ) )
		}

		@$mol_mem
		spreads(): any {
			return Object.fromEntries( [ ...this.data().keys() ].map( key => [ key, this.ShowPage( key ) ] ) )
		}

		row( id: any ) {
			const row = this.data().get( id )
			if( !row ) throw new Error( `Unknown trace ${ id }` )
			return row
		}

		override row_id( id: any ): string {
			return this.row( id ).trace ?? ''
		}

		override row_id_string( id: any ): string {
			return this.row( id ).trace ?? '-'
		}

		override row_started( id: any ): string {
			return new $mol_time_moment( this.row( id ).startedAt ).toString( 'YYYY-MM-DD hh:mm' )
		}

		override row_deliveries( id: any ): string {
			return this.row( id ).deliveries.toString()
		}

		// Units are whatever the chain's agents reported, so the cell lists them all
		// rather than assuming one number.
		override row_costs( id: any ): string {
			return $trip2g_admin_costs_text( this.row( id ).totalCosts )
		}

		override row_depth( id: any ): string {
			return this.row( id ).depthReached.toString()
		}

		// How many note versions the chain produced. Zero means every delivery in it
		// ran and stored nothing — those are hidden unless "Show empty" is on.
		override row_writes( id: any ): string {
			return this.row( id ).writes.toString()
		}
	}
}
