namespace $ {
	export class $trip2g_user_paywall_page extends $mol_object {
		
		static id(): number {
			const paywall = (window as any).__trip2g_paywall
			if (paywall) {
				return paywall.page_id
			}

			// TODO: remove it
			const el = document.getElementById('$trip2g_user_paywall.Root(1)')
			if ( el ) {
				return el.dataset.pathId ? parseInt( el.dataset.pathId, 10 ) : 0
			}

			const page_id = this.$.$mol_state_arg.value( 'page_id' )
			if( page_id ) {
				return parseInt( page_id, 10 )
			}

			throw new Error( 'Page ID not found' )
		}

		
	}
}
