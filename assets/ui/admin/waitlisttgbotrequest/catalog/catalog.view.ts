namespace $.$$ {
	export class $trip2g_admin_waitlisttgbotrequest_catalog extends $.$trip2g_admin_waitlisttgbotrequest_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminWaitListTgBotRequests {
					admin {
						allWaitListTgBotRequests {
							nodes {
								chatId
								createdAt
								notePathId
								notePath
								botName
							}
						}
					}
				}
			`)

			// Create unique key from chatId and botName combination
			return new Map( res.admin.allWaitListTgBotRequests.nodes.map( (node: any) => {
				const key = `${node.chatId}_${node.botName}`
				return [key, node]
			}))
		}

		override rows() {
			return Array.from( this.data().keys() ).map( key => this.Row( key ) )
		}

		row( key: any ) {
			return this.data().get( key )
		}

		row_chat_id( key: any ): string {
			return this.row( key ).chatId.toString()
		}

		row_bot_name( key: any ): string {
			return this.row( key ).botName
		}

		row_created_at( key: any ): string {
			const timestamp = this.row( key ).createdAt
			return new $mol_time_moment( timestamp ).toString( 'DD.MM.YYYY hh:mm:ss' )
		}

		row_note_path( key: any ): string {
			return this.row( key ).notePath
		}
	}
}