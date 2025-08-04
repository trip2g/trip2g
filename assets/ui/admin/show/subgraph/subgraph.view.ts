namespace $.$$ {
	export class $trip2g_admin_show_subgraph extends $.$trip2g_admin_show_subgraph {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminShowSubgraph($id: Int64!) {
						admin {
							subgraph(id: $id) {
								id
								name
								color
								hidden
							}
						}
					}
				`,
				{ id: this.subgraph_id() }
			)

			if (!res.admin.subgraph) {
				throw new Error('Subgraph not found')
			}

			return res.admin.subgraph
		}

		subgraph_name(): string {
			return this.data().name
		}

		@$mol_mem
		subgraph_color(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().color || ''
		}

		@$mol_mem
		override subgraph_hidden(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}

			return this.data().hidden
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation UpdateSubgraph($input: UpdateSubgraphInput!) {
						admin {
							data: updateSubgraph(input: $input) {
								... on UpdateSubgraphPayload {
									__typename
									subgraph {
										__typename
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
						hidden: this.subgraph_hidden(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'UpdateSubgraphPayload') {
				this.subgraph_color(res.admin.data.subgraph.color || '')
				// this.on_save(res.admin.data);
				return
			}

			throw new Error('Unknown response type')
		}
	}
}
