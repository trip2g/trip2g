namespace $.$$ {
	export class $trip2g_form extends $.$trip2g_form {
		override keydown( next: KeyboardEvent ) {
			if( next.keyCode === $mol_keyboard_code.enter && !this.submit_blocked() ) {
				this.submit( next )
			}
		}
	}
}
