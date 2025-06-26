namespace $.$$ {
	export class $trip2g_admin_tgbot_update extends $.$trip2g_admin_tgbot_update {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
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
				`,
				{ id: this.tgbot_id() }
			)

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
			const res = $trip2g_graphql_request(
				`
					mutation AdminUpdateTgBotMutation($input: UpdateTgBotInput!) {
						admin {
							data: updateTgBot(input: $input) {
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
				`,
				{
					input: {
						id: this.tgbot_id(),
						description: this.description(),
						enabled: this.enabled()
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'UpdateTgBotPayload') {
				this.result('TG Bot updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}