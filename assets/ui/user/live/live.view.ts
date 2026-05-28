namespace $.$$ {

	const QUERY = `
		subscription NoteChanges($filter: NoteChangesFilter!) {
			noteChanges(filter: $filter) {
				changes {
					... on NoteUpsertEvent {
						pathId
						eventType
						noteView { permalink }
					}
					... on NoteHideEvent {
						hidePathId: pathId
					}
				}
			}
		}
	`

	function is_same_path( obj: { pageId?: string, hidePathId?: string | null }, currId: number)  {
		return obj.pageId === String(currId) || obj.hidePathId === String(currId)
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
		watcher_result( next?: null ) {
			const sub = this.subscription()
			console.log('sub', sub)
			if( !sub ) return null

			const data = sub.data()
			if( !data ) {
				console.log(sub.error())
				return null
			}

			const changes: any[] = data.noteChanges?.changes ?? []
			console.log('changes', changes)
			if( changes.length === 0 ) return null

			const currentPathId = $trip2g_settings.note_path_id()

			if( this.reload_enabled() && currentPathId ) {
				const hasCurrent = changes.some( ch => is_same_path( ch, currentPathId ) )
				if( hasCurrent ) {
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
