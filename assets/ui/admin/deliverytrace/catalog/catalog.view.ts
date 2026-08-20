namespace $.$$ {
	export class $trip2g_admin_deliverytrace_catalog extends $.$trip2g_admin_deliverytrace_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_deliverytrace_catalog_list()
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
			return new $mol_time_moment( this.row( id ).startedAt ).toString( 'YYYY-MM-DD HH:mm' )
		}

		override row_deliveries( id: any ): string {
			return this.row( id ).deliveries.toString()
		}

		override row_tokens( id: any ): string {
			return this.row( id ).tokensUsed.toString()
		}

		override row_depth( id: any ): string {
			return this.row( id ).depthReached.toString()
		}
	}
}
