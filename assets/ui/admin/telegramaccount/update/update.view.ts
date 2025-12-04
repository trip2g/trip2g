namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminTelegramAccountUpdate($id: Int64!) {
			admin {
				telegramAccount(id: $id) {
					id
					phone
					displayName
					isPremium
					enabled
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminUpdateTelegramAccountMutation($input: AdminUpdateTelegramAccountInput!) {
			admin {
				payload: updateTelegramAccount(input: $input) {
					__typename
					... on AdminUpdateTelegramAccountPayload {
						account {
							id
							enabled
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramaccount_update extends $.$trip2g_admin_telegramaccount_update {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.account_id() })

			if (!res.admin.telegramAccount) {
				throw new Error('Telegram Account not found')
			}

			return res.admin.telegramAccount
		}

		account_phone(): string {
			return this.data().phone
		}

		@$mol_mem
		display_name(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().displayName || ''
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
					id: this.account_id(),
					displayName: this.display_name(),
					enabled: this.enabled()
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'AdminUpdateTelegramAccountPayload') {
				this.result('Telegram Account updated successfully')
				this.data(null)
				return
			}

			this.result('Unexpected response type')
		}
	}
}
