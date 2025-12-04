namespace $.$$ {
	const mutate = $trip2g_graphql_request( /* GraphQL */ `
		mutation AdminResetTelegramPublishNote($input: ResetTelegramPublishNoteInput!) {
			admin {
				payload: resetTelegramPublishNote(input: $input) {
					__typename
					... on ResetTelegramPublishNotePayload {
						publishNote {
							id
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegrampublishnote_button_reset extends $.$trip2g_admin_telegrampublishnote_button_reset {
		override click() {
			const res = mutate({ input: { id: this.note_path_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
