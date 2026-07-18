namespace $.$$ {
	export class $trip2g_admin_formnotes_catalog extends $.$trip2g_admin_formnotes_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_formnotes_catalog_list()
			return $trip2g_graphql_make_map( res.admin.formNotes )
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys( key => this.Content( key ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_path( id: any ): string {
			return this.row( id ).path
		}

		override row_count( id: any ): string {
			return String( this.row( id ).submitCount )
		}

		override row_last( id: any ): string {
			const ts = this.row( id ).lastSubmitAt
			if( !ts ) return '-'
			return new $mol_time_moment( ts ).toString( 'DD.MM.YYYY hh:mm:ss' )
		}

		override row_path_id( id: any ): number {
			return this.row( id ).pathId ?? 0
		}
	}
}
