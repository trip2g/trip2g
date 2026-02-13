namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminFrontmatterPatches {
				admin {
					allFrontmatterPatches {
						nodes {
							id
							description
							priority
							enabled
							includePatterns
							excludePatterns
						}
					}
				}
			}
		`
	)

	export class $trip2g_admin_frontmatterpatch_catalog extends $.$trip2g_admin_frontmatterpatch_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = data_request()

			return $trip2g_graphql_make_map( res.admin.allFrontmatterPatches.nodes )
		}

		override after_delete( id: any ) {
			this.spread( '' )
			this.data( null )
		}

		override after_create( id?: number ) {
			this.spread( `key${ id }` )
			this.data( null )
			return id || 0
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.CreateForm(),
				...this.data().mapKeys( key => this.ShowPage( key ) ),
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' && !id.startsWith( 'update/' ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_description( id: any ): string {
			return this.row( id ).description || '-'
		}

		override row_priority( id: any ): string {
			return this.row( id ).priority.toString()
		}

		override row_enabled( id: any ): string {
			return this.row( id ).enabled ? 'Yes' : 'No'
		}

		override row_patterns( id: any ): string {
			const row = this.row( id )
			const includePatterns = row.includePatterns || []
			const excludePatterns = row.excludePatterns || []

			if( includePatterns.length === 0 && excludePatterns.length === 0 ) {
				return '-'
			}

			const parts: string[] = []

			if( includePatterns.length > 0 ) {
				if( includePatterns.length === 1 ) {
					parts.push( `+${includePatterns[0]}` )
				} else {
					parts.push( `+${includePatterns[0]} (+${includePatterns.length - 1} more)` )
				}
			}

			if( excludePatterns.length > 0 ) {
				if( excludePatterns.length === 1 ) {
					parts.push( `-${excludePatterns[0]}` )
				} else {
					parts.push( `-${excludePatterns[0]} (-${excludePatterns.length - 1} more)` )
				}
			}

			return parts.join( ', ' )
		}
	}
}
