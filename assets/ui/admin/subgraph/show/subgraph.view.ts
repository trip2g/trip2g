namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
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
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation UpdateSubgraph($input: UpdateSubgraphInput!) {
			admin {
				payload: updateSubgraph(input: $input) {
					__typename
					... on UpdateSubgraphPayload {
						subgraph {
							id
							color
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_subgraph_show extends $.$trip2g_admin_subgraph_show {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.subgraph_id() })

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
			const res = mutate({
				input: {
					id: this.subgraph_id(),
					color: this.subgraph_color(),
					hidden: this.subgraph_hidden(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
			}

			if (res.admin.payload.__typename === 'UpdateSubgraphPayload') {
				this.subgraph_color(res.admin.payload.subgraph.color || '')
				// this.on_save(res.admin.payload);
				return
			}

			throw new Error('Unknown response type')
		}
	}
}
