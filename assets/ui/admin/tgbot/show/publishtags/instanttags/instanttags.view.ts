namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminTgbotShowPublishInstantTags {
				admin {
					allTelegramPublishTags {
						nodes {
							id
							label
						}
					}
				}
			}
		`
	)

	const save_request = $trip2g_graphql_request(
		`
			mutation AdminTgbotShowPublishInstantTagsSave($input: SetTgChatPublishInstantTagsInput!) {
				admin {
					data: setTgChatPublishInstantTags(input: $input) {
						__typename
						... on SetTgChatPublishInstantTagsPayload {
							success
						}
						... on ErrorPayload {
							message
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_tgbot_show_publishtags_instanttags extends $.$trip2g_admin_tgbot_show_publishtags_instanttags {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request()

			return res.admin.allTelegramPublishTags.nodes
		}

		save( chat_ids: number[] ) {
			const res = save_request( {
				input: {
					chatId: this.chat_id(),
					tagIds: chat_ids,
				},
			} )

			const { data } = res.admin

			if( data.__typename === 'ErrorPayload' ) {
				throw new Error( data.message )
			}

			if( data.__typename === 'SetTgChatPublishInstantTagsPayload' && !data.success ) {
				throw new Error( 'Unexpected response type from server' )
			}
		}

		@$mol_mem
		override value( next?: string[] ) {
			if( next !== undefined ) {
				this.save( next.map( id => parseInt( id ) ) )
			}

			return next || this.current_ids().map( id => id.toString() )
		}

		@$mol_mem
		override values(): Record<string, string> {
			const vals: Record<string, string> = {}

			this.data().forEach( s => {
				vals[ s.id ] = s.label
			} )

			return vals
		}
	}
}
