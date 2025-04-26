namespace $.$$ {
	export class $trip2g_admin_show_usersubgraphaccess extends $.$trip2g_admin_show_usersubgraphaccess {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminUserSubgraphAccess($id: Int64!) {
						admin {
							userSubgraphAccess(id: $id) {
								expiresAt
							}
						}
					}
				`,
				{ id: this.access_id() }
			)

			if (!res.admin.userSubgraphAccess) {
				throw new Error('UserSubgraphAccess not found')
			}

			return res.admin.userSubgraphAccess
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

			return next
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {
						admin {
							data: updateUserSubgraphAccess(input: $input) {
								... on UpdateUserSubgraphAccessPayload {
									userSubgraphAccess {
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
						expiresAt: this.expires_at_moment().toString(),
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
			}
		}
	}
}
