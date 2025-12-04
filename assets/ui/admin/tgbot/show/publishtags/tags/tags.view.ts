namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTgbotShowPublishTags {
			admin {
				allTelegramPublishTags {
					nodes {
						id
						label
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminTgbotShowPublishTagsSave($input: SetTgChatPublishTagsInput!) {
			admin {
				payload: setTgChatPublishTags(input: $input) {
					__typename
					... on SetTgChatPublishTagsPayload {
						success
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_tgbot_show_publishtags_tags extends $.$trip2g_admin_tgbot_show_publishtags_tags {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return res.admin.allTelegramPublishTags.nodes
		}

		save( chat_ids: number[] ) {
			const res = mutate( {
				input: {
					chatId: this.chat_id(),
					tagIds: chat_ids,
				},
			} )

			const { payload } = res.admin

			if( payload.__typename === 'ErrorPayload' ) {
				throw new Error( payload.message )
			}

			if( payload.__typename === 'SetTgChatPublishTagsPayload' && !payload.success ) {
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
