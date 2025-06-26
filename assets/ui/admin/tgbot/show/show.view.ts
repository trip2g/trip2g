namespace $.$$ {
	export class $trip2g_admin_tgbot_show extends $.$trip2g_admin_tgbot_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return super.body()
		}
	}
}