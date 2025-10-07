namespace $.$$ {
	$mol_view.roots = () => {
		return [ ...$mol_dom.document.querySelectorAll( '[mol_view_root]:not([mol_view_root=""])' ) ].map( ( node, index ) => {

			const name = node.getAttribute( 'mol_view_root' )!

			const View = ( $ as any )[ name ] as typeof $mol_view
			if( !View ) {
				$mol_fail_log( new Error( `Autobind unknown view class`, { cause: { name } } ) )
				return null
			}

			const view = View.Root( index )
			view.dom_node( node )
			return view

		} ).filter( $mol_guard_defined )
	}

	// @ts-ignore
	$mol_view.auto = () => {
		const roots = $mol_view.roots()
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
