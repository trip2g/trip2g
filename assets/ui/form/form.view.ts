namespace $.$$ {
	export class $trip2g_form extends $.$trip2g_form {
		override keydown( next: KeyboardEvent ) {
			if( next.keyCode !== $mol_keyboard_code.enter ) return
			if( next.isComposing ) return
			if( this.submit_blocked() ) return
			if( !next.ctrlKey && !$trip2g_form_enter( next.target ) ) return
			this.submit( next )
		}
	}
}
