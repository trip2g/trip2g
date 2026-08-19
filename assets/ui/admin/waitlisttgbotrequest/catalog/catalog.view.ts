namespace $.$$ {
	export class $trip2g_admin_waitlisttgbotrequest_catalog extends $.$trip2g_admin_waitlisttgbotrequest_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_waitlisttgbotrequest_catalog_list()

			// Create unique key from chatId and botName combination
			return new Map( res.admin.allWaitListTgBotRequests.nodes.map( node => {
				const key = `${node.chatId}_${node.botName}`
				return [key, node] as const
			}))
		}

		override rows() {
			return Array.from( this.data().keys() ).map( key => this.Row( key ) )
		}

		row( key: any ) {
			const row = this.data().get( key )
			if( !row ) throw new Error( 'WaitListTgBotRequest not found' )
			return row
		}

		row_chat_id( key: any ): string {
			return this.row( key ).chatId.toString()
		}

		row_bot_name( key: any ): string {
			return this.row( key ).botName
		}

		// $trip2g_admin_labeler_moment formats the value itself; formatting here
		// too fed it a rendered date it could not parse back.
		row_created_at( key: any ): string {
			return this.row( key ).createdAt
		}

		row_note_path( key: any ): string {
			return this.row( key ).notePath
		}
	}
}