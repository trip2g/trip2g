namespace $.$$ {
	const count_request = ( vars: $trip2g_graphql_AdminTelegramPublishPlanVariables ) => $trip2g_graphql_request( `
		query AdminTelegramPublishPlanCount($filter: AdminTelegramPublishNotesFilter!) {
			admin {
				allTelegramPublishNotes(filter: $filter) {
					count
				}
			}
	}`, vars ).admin.allTelegramPublishNotes.count

	const request = ( vars: $trip2g_graphql_AdminTelegramPublishPlanVariables ) => $trip2g_graphql_request( `
		query AdminTelegramPublishPlan($filter: AdminTelegramPublishNotesFilter!) {
			admin {
				allTelegramPublishNotes(filter: $filter) {
					nodes @exportType(name: "Row", single: true) {
						id
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
	}`, vars )

	export class $trip2g_admin_telegrampublishplan extends $.$trip2g_admin_telegrampublishplan {
		@$mol_mem
		data( reset?: null ) {
			const res = request( {
				filter: {
					includeSent: this.show_sent(),
					includeOutdated: this.show_outdated(),
				},
			} )

			return $trip2g_graphql_make_map( res.admin.allTelegramPublishNotes.nodes )
		}

		override show_sent_title(): string {
			const count = count_request( { filter: { includeSent: true } } )
			return super.show_sent_title().replace( '{count}', count.toString() )
		}

		override show_outdated_title(): string {
			const count = count_request( { filter: { includeOutdated: true } } )
			return super.show_outdated_title().replace( '{count}', count.toString() )
		}

		override show_sent( next?: boolean ): boolean {
			return this.$.$trip2g_state_arg.bool_value('sent', next)
		}

		override show_outdated( next?: boolean ): boolean {
			return this.$.$trip2g_state_arg.bool_value('outdated')
		}

		override rows(): readonly ( $mol_view )[] {
			return Array.from( this.data().keys() ).map( id => this.Post( id ) )
		}

		override row( id: any ) {
			return this.data().get( id )!
		}
	}

	export class $trip2g_admin_telegrampublishplan_post extends $.$trip2g_admin_telegrampublishplan_post {
		override data() {
			return $trip2g_required( super.data() )
		}

		override title(): string {
			return this.data().noteView.title
		}

		override content(): string {
			return this.data().post.content
		}

		override publish_at(): string {
			const m = new $mol_time_moment( this.data().publishAt )
			return m.toString( 'DD.MM.YYYY hh:mm' )
		}

		override seconds_until_publish() {
			return this.data().secondsUntilPublish
		}

		override warnings(): string {
			return this.data().post.warnings.join( '\n' )
		}

		override status(): string {
			return `${ super.status() } ${ this.data().status }`
		}

		override chats(): string {
			return this.data().chats.map( chat => `${ chat.chatTitle } (${ chat.chatType })` ).join( ', ' )
		}
	}
}
