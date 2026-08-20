namespace $.$$ {
	export class $trip2g_admin_labeler_paths extends $.$trip2g_admin_labeler_paths {
		// One row per path: a delivery can write several notes, and joining them
		// into one line is what made the cell unreadable.
		@$mol_mem
		override rows() {
			const paths = this.paths()
			if( !paths.length ) return [ this.Empty() ]
			return paths.map( path => this.Path( path ) )
		}

		override path( id: any ): string {
			return id
		}
	}
}
