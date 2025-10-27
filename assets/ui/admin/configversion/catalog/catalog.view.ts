namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminConfigVersions {
				admin {
					allConfigVersions {
						nodes {
							id
							createdAt
							createdBy {
								email
							}
							showDraftVersions
							defaultLayout
							timezone
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_configversion_catalog extends $.$trip2g_admin_configversion_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request()

			return $trip2g_graphql_make_map( res.admin.allConfigVersions.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.AddForm(),
				...this.data().mapKeys( key => this.Content( key ) )
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return id
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD hh:mm' )
		}

		override row_created_by( id: any ): string {
			const createdBy = this.row( id ).createdBy
			return createdBy?.email || '???'
		}

		override row_show_draft_versions( id: any ): string {
			return this.row( id ).showDraftVersions ? 'Yes' : 'No'
		}

		override row_default_layout( id: any ): string {
			return this.row( id ).defaultLayout || '-'
		}

		override row_timezone( id: any ): string {
			return this.row( id ).timezone || '-'
		}
	}
}
