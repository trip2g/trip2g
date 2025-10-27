namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminResetNotFoundPath($input: ResetNotFoundPathInput!) {
			admin {
				data: resetNotFoundPath(input: $input) {
					... on ResetNotFoundPathPayload {
						notFoundPath {
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

	export class $trip2g_admin_notfoundpath_button_reset extends $.$trip2g_admin_notfoundpath_button_reset {
		click() {
			const res = mutate({
				input: {
					id: this.notfoundpath_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}
		}
	}
}