namespace $.$$ {
	export class $trip2g_editor_pane extends $.$trip2g_editor_pane {
		override columns() {
			const cols: $mol_view[] = []

			if (this.navigator_visible()) {
				cols.push(this.Navigator())
			}

			cols.push(this.Content())

			if (this.preview_visible()) {
				cols.push(this.Preview())
			}

			return cols
		}
	}
}
