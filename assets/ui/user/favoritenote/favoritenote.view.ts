namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query FavoriteNotes {
			viewer {
				user {
					favoriteNotes {
						pathId
					}
				}
			}
		}
	`)

	const toggle_mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation ToggleFavoriteNote($input: ToggleFavoriteNoteInput!) {
			payload: toggleFavoriteNote(input: $input) {
				__typename
				... on ToggleFavoriteNotePayload {
					favoriteNotes {
						pathId
					}
				}
				... on ErrorPayload {
					message
				}
			}
		}
	`)

	export class $trip2g_user_favoritenote extends $.$trip2g_user_favoritenote {
		@$mol_mem
		path_id() {
			const sv = this.$.$mol_state_arg.value('pid')
			if (sv) {
				return parseInt(sv, 10)
			}

			const el = this.dom_node() as HTMLDivElement
			if (!el.dataset.pid) {
				throw new Error('pid not found in dataset')
			}

			return parseInt(el.dataset.pid, 10)
		}

		path_id_string() {
			return this.path_id().toString()
		}

		@$mol_mem
		favorite_note_ids(next?: number[]) {
			if (next !== undefined) {
				return next
			}

			const res = data_request()

			const notes = res.viewer.user?.favoriteNotes || []
			return notes.map(note => note.pathId)
		}

		active() {
			return this.favorite_note_ids().includes(this.path_id())
		}

		override sub() {
			if (this.active()) {
				return [this.OffIcon()]
			}

			return [this.OnIcon()]
		}

		override click() {
			const res = toggle_mutate({
				input: {
					pathId: this.path_id(),
					value: !this.active(),
				}
			})

			if (res.payload?.__typename === 'ErrorPayload') {
				throw new Error(res.payload.message)
			}

			if (res.payload?.__typename === 'ToggleFavoriteNotePayload') {
				this.favorite_note_ids(res.payload.favoriteNotes.map(note => note.pathId))
			}
		}

	}
}
