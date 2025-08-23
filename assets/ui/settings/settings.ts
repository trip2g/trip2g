namespace $ {
	const default_settings = {
		is_dev_mode: true,
	}

	const page_settings = typeof window !== 'undefined' && (window as any).__trip2g_settings || {}

	export class $trip2g_settings extends $mol_object {
		static settings(): any {
			return {
				...default_settings,
				...page_settings,
			}
		}

		static is_dev_mode(): boolean {
			return $trip2g_settings.settings().is_dev_mode || false
		}

		static dev_value( v: string ) {
			return $trip2g_settings.is_dev_mode() ? v : ''
		}
	}
}
