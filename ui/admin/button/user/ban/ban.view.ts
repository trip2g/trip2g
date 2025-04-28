namespace $.$$ {
	export class $trip2g_admin_button_user_ban extends $.$trip2g_admin_button_user_ban {
		click() {
			const res = $trip2g_graphql_request(`
				mutation AdminBanUser($input: BanUserInput!) {
					admin {
						data: banUser(input: $input) {
							... on BanUserPayload {
								user {
									id
									__typename
								}
							}

							... on ErrorPayload {
								message
							}
						}
					}
				}
			`, {
				input: {
					userId: this.user_id(),
					bannedBy: this.banned_by(),
					reason: this.reason(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}
		}
	}
}
