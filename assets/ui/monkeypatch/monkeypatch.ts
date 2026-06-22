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
			const lhash = $.$$.$trip2g_settings.locale_hash( lang ) || hash
			if( lhash ) {
				path += `?h=${ lhash }`
			}

			return JSON.parse( this.$.$mol_file.relative( path ).text().toString() )
		}


	}

	const old_make_link = $mol_state_arg.make_link

	let applied = false

	export function $trip2g_monkeypatch_apply() {
		if (applied) {
			return
		}

		$mol_locale.source = $trip2g_monkeypatch.locale_source

		$mol_state_arg.make_link = function( next: Record<string, string | null> ) {
			const r = old_make_link.call( this, next )
			return r.replace( /#!$/, '' )
		}

		console.log('Monkeypatch applied')
	}
}
