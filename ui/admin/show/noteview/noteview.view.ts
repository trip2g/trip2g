namespace $.$$ {
	export class $trip2g_admin_show_noteview extends $.$trip2g_admin_show_noteview {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(
				/* GraphQL */ `
					query AdminNoteView($id: String!) {
						admin {
							noteView(id: $id) {
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
