namespace $$ {
	// disable the title patching
	$mol_view.prototype.autorun = function() {
		try {
			this.dom_tree()
		} catch( error ) {
			$mol_fail_log( error )
		}
	}
}