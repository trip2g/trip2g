namespace $.$$ {
	export class $trip2g_admin_notfoundpattern_button_delete extends $.$trip2g_admin_notfoundpattern_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_graphql_request(`
				mutation AdminDeleteNotFoundIgnoredPattern($input: DeleteNotFoundIgnoredPatternInput!) {
					admin {
						data: deleteNotFoundIgnoredPattern(input: $input) {
							... on DeleteNotFoundIgnoredPatternPayload {
								deletedId
								__typename
							}

							... on ErrorPayload {
								message
							}
						}
					}
				}
			`, {
				input: {
					id: this.pattern_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}
		}
	}
}