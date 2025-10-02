namespace $.$$ {
	export class $trip2g_admin_gittoken_show extends $.$trip2g_admin_gittoken_show {
		override body() {
			if (this.disabled()) {
				return super.body()
			}

			return [this.DisableButton(), ...super.body()]
		}
	}
}