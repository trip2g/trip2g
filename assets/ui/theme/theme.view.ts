namespace $.$$ {
	export class $trip2g_theme extends $.$trip2g_theme {
		override render() {
			super.render()

			document.documentElement.classList.toggle('dark', !this.$.$mol_lights())
		}
	}
}
