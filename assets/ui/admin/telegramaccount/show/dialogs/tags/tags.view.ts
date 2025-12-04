namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminTelegramAccountShowDialogsTags {
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
		mutation AdminTelegramAccountShowDialogsTagsSave($input: AdminSetTelegramAccountChatPublishTagsInput!) {
			admin {
				payload: setTelegramAccountChatPublishTags(input: $input) {
					__typename
					... on AdminSetTelegramAccountChatPublishTagsPayload {
						success
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramaccount_show_dialogs_tags extends $.$trip2g_admin_telegramaccount_show_dialogs_tags {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return res.admin.allTelegramPublishTags.nodes
		}

		save( tag_ids: number[] ) {
			const res = mutate( {
				input: {
					accountId: String(this.account_id()),
					telegramChatId: this.chat_id(),
					tagIds: tag_ids.map( id => String(id) ),
				},
			} )

			const { payload } = res.admin

			if( payload.__typename === 'ErrorPayload' ) {
				throw new Error( payload.message )
			}

			if( payload.__typename === 'AdminSetTelegramAccountChatPublishTagsPayload' && !payload.success ) {
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
