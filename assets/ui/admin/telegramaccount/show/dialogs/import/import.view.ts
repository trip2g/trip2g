namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminImportTelegramAccountChannel($input: AdminImportTelegramAccountChannelInput!) {
			admin {
				payload: importTelegramAccountChannel(input: $input) {
					__typename
					... on AdminImportTelegramAccountChannelPayload {
						success
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramaccount_show_dialogs_import extends $.$trip2g_admin_telegramaccount_show_dialogs_import {
		override base_path(next?: string): string {
			return next || this.default_base_path()
		}

		submit(event?: Event) {
			const basePath = this.base_path().trim()
			if (!basePath) {
				throw new Error('Base path is required')
			}

			const res = mutate({
				input: {
					accountId: String(this.account_id()),
					channelId: this.chat_id(),
					basePath: basePath,
				},
			})

			const { payload } = res.admin

			if (payload.__typename === 'ErrorPayload') {
				throw new Error(payload.message)
			}

			this.submit_title('Import started')
		}
	}
}
