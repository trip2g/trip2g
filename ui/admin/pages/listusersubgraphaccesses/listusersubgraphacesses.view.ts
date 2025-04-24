namespace $.$$ {
	export class $trip2g_admin_pages_listusersubgraphaccesses extends $.$trip2g_admin_pages_listusersubgraphaccesses {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(/* GraphQL */ `
				query AdminListUserSubgraphAccesses {
					admin {
						data: allUserSubgraphAccesses {
							nodes {
								id
								createdAt
								expiresAt
								subgraph {
									name
								}
							}
						}
					}
				}
			`)

			const map: { [ id: string ]: typeof res.admin.data.nodes[0] } = {};

			res.admin.data.nodes.forEach( ( row ) => {
				map[ row.id ] = row
			})

			return {
				map,
				ids: Object.keys( map ),
			}
		}

		@$mol_mem
		spreads(): any {
			const pages: { [ id: string ]: any } = {};

			this.data().ids.forEach( (id) => {
				pages[id] = this.Content(id);
			});

			return pages;
		}

		row_id( id: any ): string {
			return id.toString();
		}

		row_subgraph_name( id: any ): string {
			return this.data().map[ id ].subgraph.name;
		}
	}
}
