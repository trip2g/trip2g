namespace $.$$ {
	export class $trip2g_admin_labeler_paths extends $.$trip2g_admin_labeler_paths {
		// One row per path: a delivery can write several notes, and joining them
		// into one line is what made the cell unreadable. Named path_rows, not rows:
		// $mol_labeler is itself a list, and its rows are the label and the content.
		@$mol_mem
		override path_rows() {
			const paths = this.paths()
			if( !paths.length ) return [ this.Blank() ]
			return paths.map( path => this.Path( path ) )
		}

		override path( id: any ): string {
			return id
		}
	}
}
