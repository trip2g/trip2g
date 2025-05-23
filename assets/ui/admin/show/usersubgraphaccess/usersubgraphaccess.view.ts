namespace $.$$ {
	export class $trip2g_admin_show_usersubgraphaccess extends $.$trip2g_admin_show_usersubgraphaccess {
		@$mol_mem
		all_data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
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
				`,
				{ id: this.access_id() }
			)

			return res.admin;
		}

		data() {
			const data = this.all_data()
			if (!data.userSubgraphAccess) {
				throw new Error('UserSubgraphAccess not found')
			}

			return data.userSubgraphAccess;
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
				next = new $mol_time_moment().merge(next);
			}

			return next
		}

		@$mol_mem
		subgraph_id( next?: number ): number {
			return next === undefined ? this.data().subgraphId : next
		}

		submit() {
			console.log('subgraph_id', this.subgraph_id())
			const res = $trip2g_graphql_request(
				`
					mutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {
						admin {
							data: updateUserSubgraphAccess(input: $input) {
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
				`,
				{
					input: {
						id: this.access_id(),
						expiresAt: this.expires_at_moment()?.toString('YYYY-MM-DDThh:mm:ss.sssZ') || null,
						subgraphId: this.subgraph_id(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
			}
		}
	}
}
