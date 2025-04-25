namespace $.$$ {
	export class $trip2g_admin_show_subgraph extends $.$trip2g_admin_show_subgraph {
		@$mol_mem_key
		data(id: number, reset?: null) {
			const res = $trip2g_graphql_request(
				/* GraphQL */ `
					query AdminShowSubgraph($id: Int64!) {
						admin {
							subgraph(id: $id) {
								id
								name
								color
							}
						}
					}
				`,
				{ id }
			)

			if (!res.admin.subgraph) {
				throw new Error('Subgraph not found')
			}

			return res.admin.subgraph
		}

		subgraph() {
			return this.data(this.subgraph_id())
		}

		subgraph_name(): string {
			return this.subgraph().name
		}

		@$mol_mem
		subgraph_color(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.subgraph().color || ''
		}

		submit() {
			const res = $trip2g_graphql_request(
				/* GraphQL */ `
					mutation UpdateSubgraph($input: UpdateSubgraphInput!) {
						admin {
							data: updateSubgraph(input: $input) {
								... on UpdateSubgraphPayload {
									__typename
									subgraph {
										id
										color
									}
								}
								... on ErrorPayload {
									__typename
									message
								}
							}
						}
					}
				`,
				{
					input: {
						id: this.subgraph_id(),
						color: this.subgraph_color(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'UpdateSubgraphPayload') {
				this.subgraph_color(res.admin.data.subgraph.color || '')
				this.on_save(res.admin.data);
				return
			}

			throw new Error('Unknown response type')
		}
	}
}
