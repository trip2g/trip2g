namespace $ {
	const text_types = new Set([ 'text', 'email', 'password', 'search', 'tel', 'url', 'number' ])

	export function $trip2g_form_enter( target: EventTarget | null ): boolean {
		const input = target as HTMLInputElement | null
		if( !input || input.tagName !== 'INPUT' ) return false
		if( !text_types.has( input.type ) ) return false
		if( input.closest( '[mol_pop_bubble]' ) ) return false
		return true
	}
}
