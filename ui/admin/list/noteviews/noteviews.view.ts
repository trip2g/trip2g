namespace $.$$ {
	export class $trip2g_admin_list_noteviews extends $.$trip2g_admin_list_noteviews {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(/* GraphQL */ `
				query AdminListNoteViews {
					admin {
						allNoteViews {
							nodes {
								path
								title
								free
							}
						}
					}
				}
			`)
			const map: { [ key: string ]: typeof res.admin.allNoteViews.nodes[0] } = {}
			res.admin.allNoteViews.nodes.forEach( row => {
				map[row.path] = row
			})
			return {
				map,
				ids: Object.keys(map),
			}
		}

		@$mol_mem
		spreads(): any {
			const pages: { [ id: string ]: any } = {}
			this.data().ids.forEach( id => {
				pages[id] = this.Content(id)
			})
			return pages
		}

		row_path(id: any): string {
			return this.data().map[id].path
		}
		row_title(id: any): string {
			return this.data().map[id].title
		}
		row_free(id: any): string {
			return this.data().map[id].free ? 'Yes' : 'No'
		}
	}
}
