namespace $$ {
	// disable the title patching
	$mol_view.prototype.autorun = function() {
		try {
			this.dom_tree()
		} catch( error ) {
			$mol_fail_log( error )
		}
	}

	window.addEventListener( 'popstate' , () => {
		$mol_state_arg.href( window.location.href )
	} )
}