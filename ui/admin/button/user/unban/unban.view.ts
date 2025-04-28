namespace $.$$ {
	export class $trip2g_admin_button_user_unban extends $.$trip2g_admin_button_user_unban {
		click() {
			const res = $trip2g_graphql_request(`
				mutation AdminUnbanUser($input: UnbanUserInput!) {
					admin {
						data: unbanUser(input: $input) {
							... on UnbanUserPayload {
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
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}
		}
	}
}
