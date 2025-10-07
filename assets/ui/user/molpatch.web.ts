namespace $.$$ {
	// disable the title patching
	$mol_view.auto = function() {
			const roots = this.roots()
			if( !roots.length ) return
			
			for( const root of roots ) {
				try {
					root.dom_tree()
				} catch( error ) {
					$mol_fail_log( error )
				}
			}
	}
}
