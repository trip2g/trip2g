declare global {
	interface Window {
		__trip2g_settings?: {
			title?: string
			is_dev_mode?: boolean
		}
	}
}

namespace $.$$ {
	export class $trip2g_settings extends $.$mol_object2 {
		static title() {
			return window.__trip2g_settings?.title || 'Trip2G'
		}

		// Returns value only in dev mode, empty string in production
		static dev_value<T>(value: T): T | string {
			if (location.hostname === 'localhost' || location.hostname === '127.0.0.1') {
				return value
			}
			return ''
		}
	}
}
