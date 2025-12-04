namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowTgBot($id: Int64!) {
			admin {
				tgBot(id: $id) {
					id
					name
					description
					enabled
					createdAt
					createdBy {
						email
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminUpdateTgBotMutation($input: UpdateTgBotInput!) {
			admin {
				payload: updateTgBot(input: $input) {
					__typename
					... on UpdateTgBotPayload {
						tgBot {
							id
							description
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_tgbot_update extends $.$trip2g_admin_tgbot_update {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.tgbot_id() })

			if (!res.admin.tgBot) {
				throw new Error('TG Bot not found')
			}

			return res.admin.tgBot
		}

		tgbot_name(): string {
			return this.data().name
		}

		@$mol_mem
		description(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().description || ''
		}

		@$mol_mem
		enabled(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}

			return this.data().enabled
		}

		submit() {
			const res = mutate({
				input: {
					id: this.tgbot_id(),
					description: this.description(),
					enabled: this.enabled()
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'UpdateTgBotPayload') {
				this.result('TG Bot updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}