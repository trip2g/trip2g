namespace $.$$ {

	type RenderAllRoute = {
		id: string
		label: string
		uri: string
		depth: number
	}

	const MAX_DEPTH = 2       // list -> show/new (1) -> edit (2)
	const PER_SOURCE_CAP = 3  // record/action links taken from a single iframe
	const MAX_PASSES = 4      // auto rescans while new routes keep appearing
	const RESCAN_DELAY = 1500 // ms to let freshly added iframes load before rescanning

	/**
	 * Migration smoke-test page: derives every list-level admin route from the
	 * nav catalog and renders each in an iframe, so opening this page fires every
	 * list query. Scan then reads the internals of the already-loaded iframes,
	 * discovers deeper route links (show / new / edit) that the catalogs render
	 * for their records, and appends iframes for those too - bounded by depth and
	 * a per-iframe cap so it does not explode. Errors are collected from
	 * [mol_view_error] across all iframes, discovered ones included.
	 */
	export class $trip2g_admin_debug_renderall extends $.$trip2g_admin_debug_renderall {

		base() {
			return this.$.$mol_dom_context.document.location.pathname
		}

		@ $mol_mem
		test_fail( next?: boolean ): boolean {
			if( next === undefined ) {
				return this.$.$mol_state_arg.value( 'test_fail' ) ? true : false
			}
			this.$.$mol_state_arg.value( 'test_fail', next ? '1' : null )
			return next
		}

		@ $mol_mem
		derived(): readonly RenderAllRoute[] {
			const routes: RenderAllRoute[] = []
			const base = this.base()

			const walk = ( view: $mol_view, args: Record< string, string >, labels: string[], depth: number ) => {

				// skip self to avoid recursive iframes
				if( view.constructor === this.constructor ) return

				// recurse the static nav levels only (app -> menu groups);
				// deeper catalogs are data-driven record lists whose spreads()
				// fetch data - the leaf iframe fires that query itself
				if( depth < 2 && view instanceof $mol_book2_catalog ) {
					const ids = Object.keys( view.spreads() )
					if( ids.length ) {
						for( const id of ids ) walk(
							view.Spread( id ),
							{ ... args, [ view.param() ]: id },
							[ ... labels, String( view.spread_title( id ) ) ],
							depth + 1,
						)
						return
					}
				}

				const query = Object.entries( args )
					.map( ( [ key, val ] ) => `${ key }=${ encodeURIComponent( val ) }` )
					.join( '/' )

				routes.push( {
					id: query,
					label: labels.join( ' / ' ),
					uri: `${ base }#!${ query }`,
					depth: 0,
				} )

			}

			walk( this.App(), {}, [], 0 )
			return routes
		}

		@ $mol_mem
		discovered( next?: readonly RenderAllRoute[] ): readonly RenderAllRoute[] {
			return next ?? []
		}

		@ $mol_mem
		routes(): readonly RenderAllRoute[] {
			const seen = new Set< string >()
			const all: RenderAllRoute[] = []
			for( const route of [ ... this.derived(), ... this.discovered() ] ) {
				if( seen.has( route.id ) ) continue
				seen.add( route.id )
				all.push( route )
			}
			return all
		}

		route( id: string ) {
			return this.routes().find( route => route.id === id )!
		}

		@ $mol_mem
		override cells() {
			return this.routes().map( route => this.Cell( route.id ) )
		}

		override stats() {
			const derived = this.derived().length
			const discovered = this.routes().length - derived
			const note = this.scan_note()
			return `${ this.routes().length } routes (${ derived } list + ${ discovered } discovered)${ note ? ` - ${ note }` : '' }`
		}

		@ $mol_mem
		scan_note( next = '' ): string {
			return next
		}

		override route_uri( id: string ) {
			const uri = this.route( id ).uri
			if( !this.test_fail() ) return uri
			return `${ uri }/test_fail=1`
		}

		override route_label( id: string ) {
			return this.route( id ).label
		}

		@ $mol_mem_key
		override cell_status( id: string, next = '' ): string {
			return next
		}

		frame_status( id: string ) {
			const frame = this.Frame( id ).dom_node() as HTMLIFrameElement
			try {

				const doc = frame.contentDocument
				if( !doc || !doc.body ) return 'no document'

				const errors = [ ... doc.querySelectorAll( '[mol_view_error]' ) ]
					.map( el => el.getAttribute( 'mol_view_error' )! )
					.filter( error => error !== 'Promise' )
				if( errors.length ) return `errors ${ errors.length }: ${ [ ... new Set( errors ) ].join( ', ' ) }`

				const pending = doc.querySelectorAll( '[mol_view_error="Promise"]' ).length
				if( pending ) return `loading (${ pending })`

				return 'ok'

			} catch( error ) {
				return 'inaccessible'
			}
		}

		parse_args( query: string ): Record< string, string > {
			const args: Record< string, string > = {}
			for( const chunk of query.split( '/' ) ) {
				if( !chunk ) continue
				const parts = chunk.split( '=' ).map( decodeURIComponent )
				args[ parts.shift()! ] = parts.join( '=' )
			}
			return args
		}

		hash_of( href: string ): string {
			const marker = href.indexOf( '#!' )
			return marker < 0 ? '' : href.slice( marker + 2 )
		}

		// A drill-down route keeps every arg of its source and adds at least one
		// more (row -> id, action -> id=new, show -> edit), so lateral menu links
		// (sibling spreads) and upward nav links are ignored.
		is_drilldown( source: Record< string, string >, candidate: Record< string, string > ) {
			for( const key in source ) {
				if( candidate[ key ] !== source[ key ] ) return false
			}
			return Object.keys( candidate ).length > Object.keys( source ).length
		}

		discover() {
			const base = this.base()
			const existing = new Set( this.routes().map( route => route.id ) )
			const found: RenderAllRoute[] = []
			let capped = 0

			for( const route of this.routes() ) {
				if( route.depth >= MAX_DEPTH ) continue

				const frame = this.Frame( route.id ).dom_node() as HTMLIFrameElement
				let doc: Document | null = null
				try { doc = frame.contentDocument } catch( error ) { continue }
				if( !doc || !doc.body ) continue

				const source = this.parse_args( route.id )
				let taken = 0

				for( const anchor of [ ... doc.querySelectorAll( 'a[href]' ) ] as HTMLAnchorElement[] ) {
					const hash = this.hash_of( anchor.getAttribute( 'href' ) || anchor.href )
					if( !hash || existing.has( hash ) ) continue
					if( found.some( route => route.id === hash ) ) continue

					const candidate = this.parse_args( hash )
					if( !this.is_drilldown( source, candidate ) ) continue

					if( taken >= PER_SOURCE_CAP ) { capped += 1; break }

					const added = Object.keys( candidate )
						.filter( key => source[ key ] === undefined )
						.map( key => `${ key }=${ candidate[ key ] }` )
						.join( ' ' )

					found.push( {
						id: hash,
						label: `${ route.label } / ${ added }`,
						uri: `${ base }#!${ hash }`,
						depth: route.depth + 1,
					} )
					taken += 1
				}
			}

			if( found.length ) this.discovered( [ ... this.discovered(), ... found ] )
			return { added: found.length, capped }
		}

		scan_pass( pass: number ) {
			const result = this.discover()

			for( const route of this.routes() ) {
				this.cell_status( route.id, this.frame_status( route.id ) )
			}

			this.scan_note( `pass ${ pass + 1 }: +${ result.added } discovered${ result.capped ? `, ${ result.capped } iframe(s) capped at ${ PER_SOURCE_CAP }` : '' }` )

			if( result.added > 0 && pass + 1 < MAX_PASSES ) {
				new this.$.$mol_after_timeout( RESCAN_DELAY, () => this.scan_pass( pass + 1 ) )
			}

			return this.routes().length
		}

		override scan( next?: Event | null ) {
			this.scan_pass( 0 )
			return null
		}

	}

}
