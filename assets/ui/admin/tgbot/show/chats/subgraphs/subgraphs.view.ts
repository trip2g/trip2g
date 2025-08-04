namespace $.$$ {
	export class $trip2g_admin_tgbot_show_chats_subgraphs extends $.$trip2g_admin_tgbot_show_chats_subgraphs {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(
				`
					query AdminPatreoncredentialsShowSubgraphs {
						admin {
							allSubgraphs {
								nodes {
									id
									name
								}
							}
						}
					}
				`
			)

			return res.admin.allSubgraphs.nodes
		}

		save( subgraph_ids: number[] ) {
			const res = $trip2g_graphql_request( `
				mutation AdminTgbotsShowchatsSubgraphsSave($input: SetTgChatSubgraphsInput!) {
					admin {
						data: setTgChatSubgraphs(input: $input) {
							... on SetTgChatSubgraphsPayload {
								success
							}
							... on ErrorPayload {
								message
							}
						}
					}
				}
			`, {
				input: {
					chatId: this.chat_id(),
					subgraphIds: subgraph_ids,
				},
			} )

			const { data } = res.admin

			if( data.__typename === 'ErrorPayload' ) {
				throw new Error( data.message )
			}

			if( data.__typename === 'SetTgChatSubgraphsPayload' && !data.success ) {
				throw new Error( 'Unexpected response type from server' )
			}
		}

		override sub() {
			return this.data().map( s => this.Row( s.id ) )
		}

		row( id: any ) {
			const row = this.data().find( s => s.id === id )
			if( !row ) {
				throw new Error( `Subgraph with id ${ id } not found` )
			}

			return row
		}


		@$mol_mem_key
		override item_check( id: any, next?: boolean ): boolean {
			if( next === undefined ) {
				return this.current_ids().includes( id )
			}

			const new_ids = this.data().filter( s => s.id !== id && this.item_check( s.id ) ).map( s => s.id )
			if( next ) {
				new_ids.push( id )
			}

			this.save( new_ids )

			return next
		}

		override item_title( id: any ): string {
			return this.row( id ).name
		}
	}
}