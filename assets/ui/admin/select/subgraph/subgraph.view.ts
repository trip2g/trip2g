namespace $.$$ {
	export class $trip2g_admin_select_subgraph extends $.$trip2g_admin_select_subgraph {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminSelectSubgraph {
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

			return res.admin.allSubgraphs.nodes;
		}

		dictionary(): Record<string, string> {
			const map: { [ id: number ]: string } = {};

			this.data().forEach((row) => {
				map[row.id] = row.name
			});

			return map;
		}

		@$mol_mem
		value( next?: string ): string {
			if (next === undefined) {
				const n = this.number_value()
				return n ? n.toString() : '';
			}

			if (next) {
				this.number_value(parseInt(next, 10))
			} else {
				this.number_value(null)
			}

			return next || '';
		}
	}
}
