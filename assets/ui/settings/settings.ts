namespace $ {
	export class $trip2g_settings extends $mol_object {
		static settings(): any {
			// @ts-ignore
			return typeof window !== 'undefined' && window.$trip2g || {
				is_dev_mode: true,
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