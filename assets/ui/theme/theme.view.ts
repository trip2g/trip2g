namespace $.$$ {
	export class $trip2g_theme extends $.$trip2g_theme {
		override render() {
			super.render()

			const dark = !this.$.$mol_lights()
			document.documentElement.classList.toggle('dark', dark)
			document.documentElement.classList.toggle('light', !dark)
			document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')

			// Notify non-mol widgets (e.g. the mermaid bundle) that the theme
			// changed, so they can re-render. They subscribe via
			// window.trip2g_theme_listeners.
			for (const cb of ((window as any).trip2g_theme_listeners ?? [])) {
				try { cb(dark) } catch (e) { console.error(e) }
			}
		}
	}
}
