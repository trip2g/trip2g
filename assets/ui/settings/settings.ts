namespace $.$$ {
	const settings = {
		title: 'Title Not Set',
		is_dev_mode: location.hostname === 'localhost' || location.hostname === '127.0.0.1',
		// @ts-ignore
		...(window.__trip2g_settings || {}),
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
