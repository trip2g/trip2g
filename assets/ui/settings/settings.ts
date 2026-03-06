namespace $.$$ {
	// typedef windows
	const isDevMode = typeof window === 'undefined' ||
		location.hostname === 'localhost' ||
		location.hostname === '127.0.0.1'

	const settings = {
		title: 'Title Not Set',
		is_dev_mode: isDevMode,
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
	}
}
