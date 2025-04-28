namespace $.$$ {
	export class $trip2g_admin_show_banuser extends $.$trip2g_admin_show_banuser {
		@$mol_mem
		reason(next?: string): string {
			return next ?? ''
		}

		submit() {
			const res = $trip2g_graphql_request(`
				mutation AdminBanUser($input: BanUserInput!) {
					admin {
						banUser(input: $input) {
							... on BanUserPayload {
								__typename
								user { id, __typename }
							}
							... on ErrorPayload {
								__typename
								message
							}
						}
					}
				}
			`, {
				input: {
					userId: this.$.$mol_state_arg.value('ban_id'),
					reason: this.reason(),
				},
			})

			if (res.admin.banUser.__typename === 'ErrorPayload') {
				this.result(res.admin.banUser.message)
				return
			}

			this.$.$mol_state_arg.value('ban_id', null)
		}
	}
}
