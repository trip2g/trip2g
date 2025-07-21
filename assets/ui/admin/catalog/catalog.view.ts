namespace $.$$ {
	export class $trip2g_admin_catalog extends $.$trip2g_admin_catalog {
		override menu_tools() {
			return [
				...this.actions(),
				this.close_icon(),
			]
		}
	}
}