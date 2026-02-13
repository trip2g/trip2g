namespace $.$$ {
	const delete_request = $trip2g_graphql_request(
		`
			mutation DeleteFrontmatterPatch($input: DeleteFrontmatterPatchInput!) {
				admin {
					data: deleteFrontmatterPatch(input: $input) {
						__typename
						... on DeleteFrontmatterPatchPayload {
							deletedId
						}
						... on ErrorPayload {
							error {
								code
								message
							}
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_frontmatterpatch_button_delete extends $.$trip2g_admin_frontmatterpatch_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = delete_request({
				input: {
					id: this.frontmatterpatch_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.error.message);
			}

			if (res.admin.data.__typename === 'DeleteFrontmatterPatchPayload') {
				this.after_success()
			}
		}
	}
}
