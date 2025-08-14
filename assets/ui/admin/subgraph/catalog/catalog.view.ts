namespace $.$$ {
	export class $trip2g_admin_subgraph_catalog extends $.$trip2g_admin_subgraph_catalog {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`
				query AdminListSubgraphs {
					admin {
						allSubgraphs {
							nodes {
								__typename
								id
								name
								color
								createdAt
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allSubgraphs.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key));
		}

		row(id: any) {
			return this.data().get(id);
		}

		row_id( id: any ): string {
			return this.row(id).id.toString();
		}

		row_id_number( id: any ): number {
			return this.row(id).id;
		}

		row_name( id: any ): string {
			return this.row(id).name;
		}

		row_color( id: any ): string {
			return this.row(id).color || '-'
		}
	}
}
