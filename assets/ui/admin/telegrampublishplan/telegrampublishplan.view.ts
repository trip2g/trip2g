namespace $.$$ {

	export class $trip2g_admin_telegrampublishplan extends $.$trip2g_admin_telegrampublishplan {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
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
			}`, {
				filter: {},
			} )

			return $trip2g_graphql_make_map( res.admin.allTelegramPublishNotes.nodes )
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
	}
}
