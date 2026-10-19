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
	const old_embed_load = $$.$mol_embed_native.prototype.load

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

		// $mol_embed_native waits for the next onload, so a frame whose window()
		// is recomputed after its srcdoc already loaded stays pending forever.
		$$.$mol_embed_native.prototype.load = function( frame: HTMLIFrameElement ) {
			const win = frame.contentWindow
			if( win && frame.contentDocument?.readyState === 'complete' && win.location.href !== 'about:blank' ) {
				return Promise.resolve( win )
			}
			return old_embed_load.call( this, frame )
		}

		applied = true
		console.log('Monkeypatch applied')
	}
}
