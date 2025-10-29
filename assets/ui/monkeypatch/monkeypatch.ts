namespace $ {
	let hash: string | null = ''

	const script = document.currentScript
	if( script instanceof HTMLScriptElement ) {
		const url = new URL( script.src )
		hash = url.searchParams.get( 'h' )
	}

	export class $trip2g_monkeypatch extends $mol_object {

		@$mol_mem_key
		static locale_source( lang: string ) {
			let path = `web.locale=${ lang }.json`
			if( hash ) {
				path += `?h=${ hash }`
			}

			return JSON.parse( this.$.$mol_file.relative( path ).text().toString() )
		}


	}

	export function $trip2g_monkeypatch_apply() {
		$mol_locale.source = $trip2g_monkeypatch.locale_source
	}
}
