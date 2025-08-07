namespace $.$$ {
	export class $trip2g_admin_noteview_show extends $.$trip2g_admin_noteview_show {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(
				`
					query AdminNoteView($id: String!) {
						admin {
							noteView(id: $id) {
								__typename
								path
								title
								permalink
							}
						}
					}
				`,
				{ id: this.noteview_id() }
			)

			if (!res.admin.noteView) {
				throw new Error('NoteView not found')
			}

			return res.admin.noteView;
		}

		path() {
			return this.data().path
		}

		title() {
			return this.data().title
		}

		permalink() {
			return this.data().permalink
		}
	}
}
