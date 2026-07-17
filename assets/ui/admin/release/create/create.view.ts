namespace $.$$ {
	export class $trip2g_admin_release_create extends $.$trip2g_admin_release_create {
		override submit() {
			const res = $trip2g_admin_release_create_create({
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