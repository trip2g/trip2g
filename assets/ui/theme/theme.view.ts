namespace $.$$ {
	export class $trip2g_theme extends $.$trip2g_theme {
		override render() {
			super.render()

			const dark = !this.$.$mol_lights()
			document.documentElement.classList.toggle('dark', dark)
			document.documentElement.classList.toggle('light', !dark)
			document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
		}
	}
}
