namespace $.$$ {
	export class $trip2g_admin_show_noteview extends $.$trip2g_admin_show_noteview {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(/* GraphQL */ `
				query AdminNoteView($id: String!) {
					admin {
						noteView(id: $id) {
							path
							title
							permalink
						}
					}
				}
			`, { id: this.id() })
			return res.admin.noteView || { path: '', title: '', permalink: '' }
		}
		path() { return this.data().path }
		title() { return this.data().title }
		permalink() { return this.data().permalink }
	}
}
