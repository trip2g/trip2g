namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateRelease($input: CreateReleaseInput!) {
			admin {
				payload: createRelease(input: $input) {
					__typename
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
	`)

	export class $trip2g_admin_release_create extends $.$trip2g_admin_release_create {
		override submit() {
			const res = mutate({
				input: {
					title: this.release_title(),
					homeNoteVersionId: this.home_note_version_id(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'CreateReleasePayload') {
				this.created_id( res.admin.payload.release.id )
				return
			}

			this.result('Unexpected response type')
		}
	}
}