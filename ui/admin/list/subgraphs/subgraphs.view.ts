namespace $.$$ {
	export class $trip2g_admin_list_subgraphs extends $.$trip2g_admin_list_subgraphs {
		@$mol_mem
		data() {
			this.update_marker()

			const res = $trip2g_graphql_request(/* GraphQL */ `
				query AdminListSubgraphs {
					admin {
						allSubgraphs {
							nodes {
								id
								name
								color
								createdAt
							}
						}
					}
				}
			`)

			const map: { [ id: number ]: typeof res.admin.allSubgraphs.nodes[0] } = {};

			res.admin.allSubgraphs.nodes.forEach( ( row ) => {
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
			return this.data().map[ id ].id.toString();
		}

		row_id_number( id: any ): number {
			return this.data().map[ id ].id;
		}

		row_name( id: any ): string {
			return this.data().map[ id ].name;
		}

		row_color( id: any ): string {
			return this.data().map[ id ].color || '-'
		}
	}
}
