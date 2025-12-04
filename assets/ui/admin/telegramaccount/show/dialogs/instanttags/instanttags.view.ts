namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminTelegramAccountShowDialogsInstantTags {
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
			mutation AdminTelegramAccountShowDialogsInstantTagsSave($input: AdminSetTelegramAccountChatPublishInstantTagsInput!) {
				admin {
					data: setTelegramAccountChatPublishInstantTags(input: $input) {
						__typename
						... on AdminSetTelegramAccountChatPublishInstantTagsPayload {
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

	export class $trip2g_admin_telegramaccount_show_dialogs_instanttags extends $.$trip2g_admin_telegramaccount_show_dialogs_instanttags {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request()

			return res.admin.allTelegramPublishTags.nodes
		}

		save( tag_ids: number[] ) {
			const res = save_request( {
				input: {
					accountId: String(this.account_id()),
					telegramChatId: this.chat_id(),
					tagIds: tag_ids.map( id => String(id) ),
				},
			} )

			const { data } = res.admin

			if( data.__typename === 'ErrorPayload' ) {
				throw new Error( data.message )
			}

			if( data.__typename === 'AdminSetTelegramAccountChatPublishInstantTagsPayload' && !data.success ) {
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

			this.data().forEach( (s: any) => {
				vals[ s.id ] = s.label
			} )

			return vals
		}
	}
}
