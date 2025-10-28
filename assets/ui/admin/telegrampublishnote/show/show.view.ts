namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTelegramPublishNote($id: Int64!) {
			admin {
				telegramPublishNote(id: $id) {
					id
					createdAt
					publishAt
					secondsUntilPublish
					publishedAt
					status
					tags {
						label
					}
					chats {
						chatTitle
						chatType
					}
					noteView {
						title
					}
					post {
						content
						warnings
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegrampublishnote_show extends $.$trip2g_admin_telegrampublishnote_show {
		@$mol_mem
		data( reset?: null ) {
			const res = request( { id: this.telegrampublishnote_id() } ).admin.telegramPublishNote
			if (!res) throw new Error('not found')
			return res
		}

		override tools() {
			const items: $mol_view[] = []

			if (this.data().status === 'Sent') {
				items.push( this.ResetButton() )
			} else {
				items.push( this.SendButton() )
			}

			return items
		}

		override note_title(): string {
			return this.data().noteView.title
		}

		override created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'DD.MM.YYYY hh:mm' )
		}

		override publish_at(): string {
			const m = new $mol_time_moment( this.data().publishAt )
			return m.toString( 'DD.MM.YYYY hh:mm' )
		}

		override status(): string {
			return this.data().status
		}

		override chats(): string {
			return this.data().chats.map( chat => `${ chat.chatTitle } (${ chat.chatType })` ).join( ', ' )
		}

		override seconds_until_publish(): number {
			return this.data().secondsUntilPublish
		}

		override warnings(): string {
			return this.data().post.warnings.join( '\n' ) || super.warnings()
		}

		override content_text(): string {
			return this.data().post.content
		}
	}
}
