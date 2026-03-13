namespace $.$$ {
	// typedef windows
	const isDevMode = typeof window === 'undefined' ||
		location.hostname === 'localhost' ||
		location.hostname === '127.0.0.1'

	const settings = {
		title: 'Title Not Set',
		is_dev_mode: isDevMode,
		ui_lang: '',
		note_lang: '',
		// @ts-ignore
		...(typeof window !== 'undefined' ? window.__trip2g_settings : {}),
	}

	export class $trip2g_settings extends $.$mol_object2 {
		static title() {
			return settings.title
		}

		// Returns value only in dev mode, empty string in production
		static dev_value<T>(value: T): T | string {
			return settings.is_dev_mode ? value : ''
		}

		static ui_lang() {
			return settings.ui_lang
		}

		static note_lang() {
			return settings.note_lang
		}

		static set_lang(lang: string) {
			if (!settings.ui_lang || settings.ui_lang === lang) return lang

			const url = new URL(location.href)
			url.searchParams.set('setlang', lang)
			if (url.href !== location.href) {
				location.href = url.href
			}

			return lang
		}
	}
}
