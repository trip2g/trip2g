namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTgBotInviteChats($filter: AdminTgBotChatsFilterInput!) {
			admin {
				tgBotChats(filter: $filter) {
					nodes {
						id
						chatType
						chatTitle
						addedAt
						removedAt
						memberCount
						subgraphInvites {
							id
							subgraphId
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_tgbot_show_invitechats extends $.$trip2g_admin_tgbot_show_invitechats {
		@$mol_mem
		data( reset?: null ) {
			const res = request({
				filter: {
					botId: this.bot_id(),
				}
			})

			return $trip2g_graphql_make_map( res.admin.tgBotChats.nodes )
		}

		@$mol_mem
		override data_rows() {
			return this.data().map( id => this.Row( id ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_chat_type( id: any ): string {
			return this.row( id ).chatType
		}

		override row_chat_title( id: any ): string {
			return this.row( id ).chatTitle || '-'
		}

		override row_member_count( id: any ): string {
			return this.row( id ).memberCount.toString()
		}

		row_added_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).addedAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		row_removed_at( id: any ): string {
			const removedAt = this.row( id ).removedAt
			if( !removedAt ) return '-'
			const m = new $mol_time_moment( removedAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		override row_subgraph_ids( id: any ) {
			return this.row(id).subgraphInvites.map( a => a.subgraphId)
		}
	}
}
