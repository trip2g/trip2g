namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreateUserSubgraphAccess($input: CreateUserSubgraphAccessInput!) {
			admin {
				data: createUserSubgraphAccess(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on CreateUserSubgraphAccessPayload {
						accesses {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_usersubgraphaccesses_create extends $.$trip2g_admin_usersubgraphaccesses_create {
		@$mol_mem
		user_id(next?: number): number {
			if (next !== undefined) return next
			const id = this.$.$mol_state_arg.value('user_id')
			return id ? parseInt(id, 10) : 0
		}

		@$mol_mem
		subgraph_id(next?: number): number {
			return next ?? 0
		}

		@$mol_mem
		expires_at_moment(next?: any) {
			if (next === undefined) {
				return null
			}

			if (next) {
				next = new $mol_time_moment().merge(next)
			}

			return next
		}

		override submit() {
			const userId = this.user_id()
			if (!userId) {
				throw new Error('User ID is required')
			}

			const subgraphId = this.subgraph_id()
			if (!subgraphId) {
				throw new Error('Subgraph is required')
			}

			const res = mutate({
				input: {
					userId: userId,
					subgraphIds: [subgraphId],
					expiresAt: $trip2g_moment_toserver(this.expires_at_moment()),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateUserSubgraphAccessPayload') {
				const accessId = res.admin.data.accesses[0]?.id
				if (accessId) {
					this.$.$mol_state_arg.value('id', accessId.toString())
				}
				this.result('Access created successfully')
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}
