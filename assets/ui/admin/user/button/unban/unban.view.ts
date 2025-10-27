namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminUnbanUser($input: UnbanUserInput!) {
			admin {
				payload: unbanUser(input: $input) {
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
	`)
	export class $trip2g_admin_user_button_unban extends $.$trip2g_admin_user_button_unban {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = mutate({
				input: {
					userId: this.user_id(),
				},
			});

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message);
			}
		}
	}
}
