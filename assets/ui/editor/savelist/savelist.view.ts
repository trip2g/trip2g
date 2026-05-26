namespace $.$$ {
	export class $trip2g_editor_savelist extends $.$trip2g_editor_savelist {
		override file_rows(): readonly $mol_view[] {
			return this.files().map(path => this.File_row(path))
		}

		override file_label(path: string): string {
			return path
		}

		override selected(): string[] {
			return this.files().filter(path => this.checked(path))
		}
	}
}
