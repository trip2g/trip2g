namespace $.$$ {
	type NoteKey = string | number

	export class $trip2g_reader extends $.$trip2g_reader {

		@$mol_mem_key
		note( key: NoteKey, reset?: null ) {
			let path: string | undefined
			let pathId: number | undefined

			if ( typeof key === 'string' ) {
				path = key.replace( '?version=latest', '' )
			} else if (typeof key === 'number') {
				pathId = key
			}

			const res = $trip2g_graphql_request( `
				query ReaderQuery($input: NoteInput!) {
					note(input: $input) {
						title
						html
						pathId
					}
				}
			`, {
				input: { path, pathId, referer: "" },
			} )

			if( !res.note ) {
				throw new Error( `Note not found for path: ${ key }` )
			}

			return res.note
		}

		@$mol_mem
		first_page_path() {
			return this.$.$mol_state_arg.value( 'path' ) || '/'
		}

		@$mol_mem
		path_ids( next?: number[] ) {
			return next || []
		}

		note_keys() {
			return [
				this.first_page_path(),
				...this.path_ids(),
			]
		}

		override close_click( key: NoteKey ) {
			const path_ids = this.path_ids()
			this.path_ids( path_ids.filter( id => id !== key ) )
		}

		override body(): readonly ( $mol_view )[] {
			return this.note_keys().map( key => this.Note( key ) )
		}

		override content_title( key: NoteKey ) {
			return this.note( key ).title
		}

		override content_html( key: NoteKey ) {
			return this.note( key ).html
		}

		override handle_next_url( path: string, next?: string | undefined ) {
			if( next ) {
				if (typeof next !== 'number') {
					return ''
				}

				const path_ids = [ ...this.path_ids() ]
				const index = path_ids.indexOf( next )
				if( index === -1 ) {
					path_ids.push( next )
				}
				this.path_ids( path_ids )

				new $mol_after_timeout( 100, () => {
					this.Note( next ).dom_node().scrollIntoView( {
						behavior: 'smooth',
						block: 'start',
						inline: 'nearest',
					} )
				} )
			}

			return next || ''
		}
	}

	export class $trip2g_reader_html extends $.$trip2g_reader_html {
		// override link_uri( id: any ): string {
		// 	const uri = super.link_uri( id )
		// 	console.log(uri)
		// 	return uri
		// }

		override link_click( el: any, e: MouseEvent ) {
			if( this.handle_next ) {
				e.preventDefault()
				const key = parseInt( el.dataset.pid, 10 ) || el.getAttribute('href')
				this.handle_next( key )
			}
		}
	}
}