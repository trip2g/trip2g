namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminDeleteNotFoundIgnoredPattern($input: DeleteNotFoundIgnoredPatternInput!) {
			admin {
				payload: deleteNotFoundIgnoredPattern(input: $input) {
					__typename
					... on DeleteNotFoundIgnoredPatternPayload {
						deletedId
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_notfoundpattern_button_delete extends $.$trip2g_admin_notfoundpattern_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = mutate({
				input: {
					id: this.pattern_id(),
				},
			});

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message);
			}
		}
	}
}