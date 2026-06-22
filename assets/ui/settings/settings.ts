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
		js_urls: [] as string[],
		locale_hashes: {} as Record<string, string>,
		note_path: '',
		note_path_id: 0,
		note_version_id: '',
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

		static js_urls(): string[] {
			return settings.js_urls
		}

		static locale_hash(lang: string): string {
			return settings.locale_hashes[lang] || ''
		}

		static note_path() {
			return settings.note_path
		}

		static note_path_id() {
			return settings.note_path_id
		}

		static note_version_id() {
			return settings.note_version_id
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
