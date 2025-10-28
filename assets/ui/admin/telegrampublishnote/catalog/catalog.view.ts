namespace $.$$ {
	const count_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTelegramPublishNoteCount($filter: AdminTelegramPublishNotesFilter!) {
			admin {
				allTelegramPublishNotes(filter: $filter) {
					count
				}
			}
		}
	`)

	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTelegramPublishNotes($filter: AdminTelegramPublishNotesFilter!) {
			admin {
				allTelegramPublishNotes(filter: $filter) {
					nodes {
						id
						publishAt
						secondsUntilPublish
						publishedAt
						status
						noteView {
							title
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegrampublishnote_catalog extends $.$trip2g_admin_telegrampublishnote_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request({
				filter: {
					includeSent: this.show_sent(),
					includeOutdated: this.show_outdated(),
				},
			})

			return $trip2g_graphql_make_map( res.admin.allTelegramPublishNotes.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: null, // No add form for this catalog
				...this.data().mapKeys( key => this.Content( key ) )
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return id
		}

		override show_sent_title(): string {
			const res = count_request({ filter: { includeSent: true } })
			const count = res.admin.allTelegramPublishNotes.count
			return super.show_sent_title().replace( '{count}', count.toString() )
		}

		override show_outdated_title(): string {
			const res = count_request({ filter: { includeOutdated: true } })
			const count = res.admin.allTelegramPublishNotes.count
			return super.show_outdated_title().replace( '{count}', count.toString() )
		}

		// override show_sent( next?: boolean ): boolean {
		// 	return this.$.$trip2g_state_arg.bool_value('sent', next)
		// }

		// override show_outdated( next?: boolean ): boolean {
		// 	return this.$.$trip2g_state_arg.bool_value('outdated', next)
		// }

		override row_title( id: any ): string {
			return this.row( id ).noteView.title
		}

		override row_publish_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).publishAt )
			return m.toString( 'DD.MM.YYYY hh:mm' )
		}

		override row_status( id: any ): string {
			return this.row( id ).status
		}

		override row_seconds_until_publish( id: any ): string {
			const seconds = this.row( id ).secondsUntilPublish
			if (seconds <= 0) return 'Ready'
			
			const minutes = Math.floor(seconds / 60)
			const hours = Math.floor(minutes / 60)
			const days = Math.floor(hours / 24)
			
			if (days > 0) return `${days}d ${hours % 24}h`
			if (hours > 0) return `${hours}h ${minutes % 60}m`
			return `${minutes}m`
		}
	}
}
