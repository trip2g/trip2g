namespace $.$$ {
	export class $trip2g_admin_noteview_warnings extends $.$trip2g_admin_noteview_warnings {

		@$mol_mem
		data() {
			const res = $trip2g_graphql_request( `
				query AdminNoteWarnings($filter: AdminLatestNoteViewsFilter) {
					admin {
						allLatestNoteViews(filter: $filter) {
							nodes {
								id
								path
								warnings {
									level
									message
								}
							}
						}
					}
				}
			`, {
				filter: {
					withWarnings: true,
				},
			} )

			return $trip2g_graphql_make_map( res.admin.allLatestNoteViews.nodes )
		}

		override rows(): readonly ( any )[] {
			return this.data().map( key => this.Row( key ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): string {
			return this.row( id ).id
		}

		override row_path( id: any ): string {
			return this.row( id ).path
		}

		override row_warnings( id: any ) {
			return this.row( id ).warnings.map( w => this.WarningRow( `${ w.level }: ${ w.message }` ) )
		}

		override warning_content( id: any ): string {
			return id
		}
	}
}