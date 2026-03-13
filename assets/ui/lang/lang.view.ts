namespace $.$$ {
	export class $trip2g_lang extends $.$trip2g_lang {
		@$mol_mem
		current_lang() {
			const lang = this.$.$mol_locale.lang()
			this.$.$trip2g_settings.set_lang(lang)
			return lang
		}

		override render() {
			super.render()
			this.current_lang()
		}
	}
}
