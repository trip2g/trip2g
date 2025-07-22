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

		override row_path( id: any ): string {
			return this.row( id ).path
		}

		@$mol_mem_key
		more_opened( id: any, next?: boolean ) {
			return next || false
		}

		override warning_more_toggle( id: any ) {
			this.more_opened( id, !this.more_opened( id ) )
		}

		override warning_more_title( id: any ): string {
			return this.more_opened( id ) ? 'Show less' : `Show all (${ this.row( id ).warnings.length })`
		}

		override row_warnings( id: any ) {
			const { warnings } = this.row( id )
			const limit = 2

			let items: $mol_view[] = warnings.map( ( _, idx ) => this.WarningRow( `${ id }:${ idx }` ) )

			if ( !this.more_opened( id ) ) {
				items = items.slice( 0, limit )
			}

			if( warnings.length > limit ) {
				items.push( this.MoreButton( id ) )
			}

			return items
		}

		override warning_content( id: any ): string {
			const [ row_id, warning_id ] = id.split( ':' )
			const w = this.row( row_id ).warnings[ +warning_id ]

			return `${ w.level }: ${ w.message }`
		}
	}
}