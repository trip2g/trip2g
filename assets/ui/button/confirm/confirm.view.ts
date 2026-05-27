namespace $.$$ {
	export class $trip2g_button_confirm extends $.$trip2g_button_confirm {
		@$mol_mem
		asking( next?: boolean ): boolean { return next ?? false }

		buttons() {
			return this.asking() ? [ this.ConfirmButton() ] : [ this.ActionButton() ]
		}

		ask( next?: null ) {
			if( next === undefined ) return null
			this.asking( true )
			return null
		}

		execute( next?: null ) {
			if( next === undefined ) return null
			this.asking( false )
			this.click( next )
			return null
		}
	}
}
