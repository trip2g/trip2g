namespace $.$$ {

	const QUERY = `
		subscription NoteChanges($filter: NoteChangesFilter!) {
			noteChanges(filter: $filter) {
				changes {
					... on NoteUpsertEvent {
						pathId
						eventType
						changedHtmlSelectors
						noteView { permalink }
					}
					... on NoteHideEvent {
						hidePathId: pathId
					}
				}
			}
		}
	`

	const HIGHLIGHT_KEY = '--note-highlight'

	function is_same_path( obj: { pathId?: string, hidePathId?: string | null }, currId: number ) {
		return String( obj.pathId ?? obj.hidePathId ) === String( currId )
	}

	export class $trip2g_user_live extends $.$trip2g_user_live {

		reload_enabled( next?: boolean ) {
			return this.$.$trip2g_user_live_reload_toggler.value( next )
		}

		follow_enabled( next?: boolean ) {
			return this.$.$trip2g_user_live_follow_toggler.value( next )
		}

		@ $mol_mem
		subscription() {
			if( !this.reload_enabled() && !this.follow_enabled() ) return null
			return $trip2g_graphql_raw_subscription( QUERY, {
				filter: { includePatterns: [ '**/*.md' ] }
			} )
		}

		@ $mol_mem
		highlight_on_load( next?: null ) {
			const selector = sessionStorage.getItem( HIGHLIGHT_KEY )
			if( !selector ) return null
			sessionStorage.removeItem( HIGHLIGHT_KEY )
			requestAnimationFrame( () => {
				const el = document.querySelector( selector )
				if( !el ) return
				el.classList.add( '--note-changed' )
				el.scrollIntoView( { behavior: 'smooth', block: 'center' } )
			} )
			return null
		}

		@ $mol_mem
		watcher_result( next?: null ) {
			const sub = this.subscription()
			if( !sub ) return null

			const data = sub.data()
			if( !data ) {
				const err = sub.error()
				if( err ) console.log( 'subscription error', err )
				return null
			}

			const changes: any[] = data.noteChanges?.changes ?? []
			if( changes.length === 0 ) return null

			const currentPathId = $trip2g_settings.note_path_id()
			console.log(JSON.stringify(changes, null, 2))

			if( this.reload_enabled() && currentPathId ) {
				const match = changes.find( ch => is_same_path( ch, currentPathId ) )
				if( match ) {
					const selector: string | undefined = match.changedHtmlSelectors?.[ 0 ]
					if( selector ) sessionStorage.setItem( HIGHLIGHT_KEY, selector )
					setTimeout( () => location.reload(), 0 )
					return null
				}
			}

			if( this.follow_enabled() ) {
				const first = changes[ 0 ]
				const permalink = first?.noteView?.permalink
				if( permalink ) setTimeout( () => { location.href = permalink }, 0 )
			}

			return null
		}

	}

}
