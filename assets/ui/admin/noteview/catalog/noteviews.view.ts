namespace $.$$ {
	export class $trip2g_admin_noteview_catalog extends $.$trip2g_admin_noteview_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(`
				query AdminListNoteViews {
					admin {
						allLatestNoteViews {
							nodes {
								id
								path
								title
								free
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allLatestNoteViews.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key));
		}

		row( id: any ) {
			return this.data().get(id);
		}

		row_id( id: any ): string {
			return this.row(id).id;
		}

		row_path( id: any ): string {
			return this.row(id).path;
		}

		row_title( id: any ): string {
			return this.row(id).title;
		}

		row_free( id: any ): string {
			return this.row(id).free ? 'Yes' : 'No';
		}
	}
}
