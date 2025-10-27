namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminSelectSubgraphList {
			admin {
				allSubgraphs {
					nodes {
						id
						name
					}
				}
			}
		}
	`)

	export class $trip2g_admin_subgraph_select_list extends $.$trip2g_admin_subgraph_select_list {
		@$mol_mem
		data( reset?: null ) {
			const res = request()

			return res.admin.allSubgraphs.nodes
		}

		dictionary(): Record<string, string> {
			const map: { [ id: number ]: string } = {}

			this.data().forEach( ( row ) => {
				map[ row.id ] = row.name
			} )

			return map
		}

		@$mol_mem
		override value( next?: string[] ) {
			if( next === undefined ) {
				return this.ids().map( id => id.toString() )
			}

			this.ids( next.map( id => parseInt( id, 10 ) ) )

			return next || []
		}
	}
}