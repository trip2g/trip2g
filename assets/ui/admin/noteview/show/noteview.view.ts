namespace $.$$ {
	export class $trip2g_admin_noteview_show extends $.$trip2g_admin_noteview_show {
		@$mol_mem
		data() {
			const res = $trip2g_admin_noteview_show_data({ id: this.noteview_id() })

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

		override content() {
			return this.data().content
		}
	}
}
