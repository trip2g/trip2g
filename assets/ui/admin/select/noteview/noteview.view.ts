namespace $.$$ {
	export class $trip2g_admin_select_noteview extends $.$trip2g_admin_select_noteview {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(
				`
					query AdminSelectNoteView {
						admin {
							allLatestNoteViews {
								nodes {
									versionId
									path
									title
								}
							}
						}
					}
				`
			)

			return res.admin.allLatestNoteViews.nodes
		}

		dictionary(): Record<string, string> {
			const map: { [ id: string ]: string } = {
				'': 'Without note view',
			}

			this.data().forEach( ( row ) => {
				map[ row.versionId ] = `${ row.title } (${ row.path })`
			} )

			return map
		}

		@$mol_mem
		value( next?: string ): string {
			if( next === undefined ) {
				const v = this.version_id()
				return v ? v.toString() : ''
			}

			if( next ) {
				this.version_id( parseInt( next, 10 ) )
			} else {
				this.version_id( null )
			}

			return next || ''
		}
	}
}