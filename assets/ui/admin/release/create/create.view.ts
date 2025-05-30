namespace $.$$ {
	export class $trip2g_admin_release_create extends $.$trip2g_admin_release_create {
		override submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateRelease($input: CreateReleaseInput!) {
						admin {
							data: createRelease(input: $input) {
								... on CreateReleasePayload {
									release {
										id
									}
								}
								... on ErrorPayload {
									message
								}
							}
						}
					}
				`,
				{
					input: {
						title: this.release_title(),
						homeNoteVersionId: this.home_note_version_id(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateReleasePayload') {
				this.created_id( res.admin.data.release.id )
				return
			}

			this.result('Unexpected response type')
		}
	}
}