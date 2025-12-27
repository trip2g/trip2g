namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminUserSubgraphAccess($id: Int64!) {
			admin {
				allSubgraphs {
					nodes {
						id
						name
					}
				}

				userSubgraphAccess(id: $id) {
					userId
					subgraphId
					expiresAt
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {
			admin {
				payload: updateUserSubgraphAccess(input: $input) {
					... on UpdateUserSubgraphAccessPayload {
						userSubgraphAccess {
							__typename
							expiresAt
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_usersubgraphaccesses_show extends $.$trip2g_admin_usersubgraphaccesses_show {
		@$mol_mem
		all_data(reset?: null) {
			const res = request({ id: this.access_id() })

			return res.admin
		}

		data() {
			const data = this.all_data()
			if (!data.userSubgraphAccess) {
				throw new Error('UserSubgraphAccess not found')
			}

			return data.userSubgraphAccess
		}

		@$mol_mem
		expires_at_moment(next?: any) {
			if (next === undefined) {
				const raw = this.data().expiresAt

				if (raw) {
					return new $mol_time_moment(raw)
				}

				return null
			}

			if (next) {
				next = new $mol_time_moment().merge(next)
			}

			return next
		}

		@$mol_mem
		subgraph_id(next?: number): number {
			return next === undefined ? this.data().subgraphId : next
		}

		submit() {
			const res = mutate({
				input: {
					id: this.access_id(),
					expiresAt: $trip2g_moment_toserver(this.expires_at_moment()),
					subgraphId: this.subgraph_id(),
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
			}
		}
	}
}
