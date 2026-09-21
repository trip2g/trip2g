namespace $ {

	/**
	 * Which sign-in methods this instance offers, asked for the current page so
	 * the OAuth URLs carry the right return path. Null means the question could
	 * not be answered: callers fall back to showing everything rather than
	 * locking the visitor out of a door that may well be open.
	 */
	export function $trip2g_auth_methods_here(): trip2g_auth_methodsQuery | null {
		const here = new URL( $$.$mol_state_arg.href() )
		const redirectUrl = here.pathname + here.search + here.hash
		try {
			return $trip2g_auth_methods({ input: { redirectUrl } })
		} catch( error ) {
			if( $mol_promise_like( error ) ) $mol_fail_hidden( error )
			return null
		}
	}

}
