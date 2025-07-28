namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_show_subgraphs extends $.$trip2g_admin_patreoncredentials_show_subgraphs {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(
				`
					query AdminPatreoncredentialsShowSubgraphs {
						admin {
							allSubgraphs {
								nodes {
									id
									name
								}
							}
						}
					}
				`
			)

			return res.admin.allSubgraphs.nodes
		}

		save( subgraph_ids: number[] ) {
			const res = $trip2g_graphql_request( `
				mutation AdminPatreoncredentialsShowSubgraphsSave($input: SetPatreonTierSubgraphsInput!) {
					admin {
						data: setPatreonTierSubgraphs(input: $input) {
							... on SetPatreonTierSubgraphsPayload {
								success
							}
							... on ErrorPayload {
								message
							}
						}
					}
				}
			`, {
				input: {
					tierId: this.tier_id(),
					subgraphIds: subgraph_ids,
				},
			} )

			const { data } = res.admin

			if( data.__typename === 'ErrorPayload' ) {
				throw new Error( data.message )
			}

			if( data.__typename === 'SetPatreonTierSubgraphsPayload' && !data.success ) {
				throw new Error( 'Unexpected response type from server' )
			}
		}

		override sub() {
			return this.data().map( s => this.Row( s.id ) )
		}

		row( id: any ) {
			const row = this.data().find( s => s.id === id )
			if( !row ) {
				throw new Error( `Subgraph with id ${ id } not found` )
			}

			return row
		}


		@$mol_mem_key
		override item_check( id: any, next?: boolean ): boolean {
			if( next === undefined ) {
				return this.current_ids().includes( id )
			}

			const new_ids = this.data().filter( s => s.id !== id && this.item_check( s.id ) ).map( s => s.id )
			if( next ) {
				new_ids.push( id )
			}

			this.save( new_ids )

			return next
		}

		override item_title( id: any ): string {
			return this.row( id ).name
		}
	}
}