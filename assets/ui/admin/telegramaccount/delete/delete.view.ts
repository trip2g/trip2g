namespace $.$$ {
	const delete_request = $trip2g_graphql_request(
		`
			mutation AdminDeleteTelegramAccount($input: AdminDeleteTelegramAccountInput!) {
				admin {
					data: deleteTelegramAccount(input: $input) {
						... on AdminDeleteTelegramAccountPayload {
							success
							__typename
						}

						... on ErrorPayload {
							message
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_telegramaccount_delete extends $.$trip2g_admin_telegramaccount_delete {
		delete() {
			const res = delete_request({
				input: {
					id: String(this.account_id()),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'AdminDeleteTelegramAccountPayload') {
				this.$.$mol_state_arg.value('id', null)
				this.$.$mol_state_arg.value('action', null)
			}
		}
	}
}
