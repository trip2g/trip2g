namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminListNoteViews {
			admin {
				allLatestNoteViews {
					nodes {
						id
						path
						title
						free
						permalink
					}
				}
			}
		}
	`)

	export class $trip2g_admin_noteview_catalog extends $.$trip2g_admin_noteview_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map(res.admin.allLatestNoteViews.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key));
		}

		row( id: any ) {
			return this.data().get(id);
		}

		override row_id( id: any ): string {
			return this.row(id).id;
		}

		override row_path( id: any ): string {
			return this.row(id).path;
		}

		override row_title( id: any ): string {
			return this.row(id).title;
		}

		override row_free( id: any ): string {
			return this.row(id).free ? 'Yes' : 'No';
		}

		override row_permalink_el( id: any ) {
			const l = this.row(id).permalink;
			if (l.includes('/_')) {
				return this.NoPermalink(id)
			}
			return this.Permalink(id)
		}

		override row_permalink( id: any ): string {
			return this.row(id).permalink;
		}
	}
}
