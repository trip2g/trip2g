namespace $.$$ {

	export class $trip2g_user_live_reload_toggler extends $.$trip2g_user_live_reload_toggler {

		static value(next?: boolean) {
			return this.$.$mol_state_local.value( 'trip2g_live_reload', next ) ?? false
		}

		@ $mol_mem
		checked( next?: boolean | undefined ) {
			return $trip2g_user_live_reload_toggler.value( next )
		}

		override title() {
			return this.labels()[$trip2g_bool.on_off(this.checked())]
		}

	}

}
