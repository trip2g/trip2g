namespace $.$$ {
	export class $trip2g_admin_noteasset_catalog extends $.$trip2g_admin_noteasset_catalog {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`
				query AdminNoteAssets {
					admin {
						allLatestNoteAssets {
							nodes {
								id
								absolutePath
								fileName
								size
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allLatestNoteAssets.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.ShowPage(key))
		}

		row(id: any) {
			return this.data().get(id)
		}

		override row_id(id: any): number {
			return this.row(id).id
		}

		override row_id_string(id: any): string {
			return this.row(id).id.toString()
		}

		override row_absolute_path(id: any): string {
			return this.row(id).absolutePath
		}

		override row_size_formatted(id: any): string {
			const size = this.row(id).size
			if (size < 1024) return `${size} B`
			if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
			if (size < 1024 * 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
			return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`
		}
	}
}
