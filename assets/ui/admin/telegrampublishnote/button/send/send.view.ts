namespace $.$$ {
	const mutate = $trip2g_graphql_request( /* GraphQL */ `
		mutation AdminSendTelegramPublishNoteNow($input: SendTelegramPublishNoteNowInput!) {
			admin {
				payload: sendTelegramPublishNoteNow(input: $input) {
					... on SendTelegramPublishNoteNowPayload {
						success
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegrampublishnote_button_send extends $.$trip2g_admin_telegrampublishnote_button_send {
		override click() {
			const res = mutate({ input: { id: this.note_path_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
