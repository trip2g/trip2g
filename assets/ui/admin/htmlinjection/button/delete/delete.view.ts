namespace $.$$ {
	export class $trip2g_admin_htmlinjection_button_delete extends $.$trip2g_admin_htmlinjection_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_graphql_request(`
				mutation AdminDeleteHtmlInjection($input: DeleteHtmlInjectionInput!) {
					admin {
						data: deleteHtmlInjection(input: $input) {
							... on DeleteHtmlInjectionPayload {
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
					id: this.htmlinjection_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}

			if (res.admin.data.__typename === 'DeleteHtmlInjectionPayload') {
				this.after_success()
			}
		}
	}
}
