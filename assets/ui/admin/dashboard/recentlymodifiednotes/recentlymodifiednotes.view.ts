namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminRecentlyModifiedNotes {
			admin {
				recentlyModifiedNoteViews {
				id
				title
				permalink
				}
			}
		}
	`)

	export class $trip2g_admin_dashboard_recentlymodifiednotes extends $.$trip2g_admin_dashboard_recentlymodifiednotes {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return $trip2g_graphql_make_map( res.admin.recentlyModifiedNoteViews )
		}

		override rows() {
			console.log(this.data())
			return this.data().map( id => this.Row( id ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_title( id: any ): string {
			return this.row( id ).title
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
