namespace $.$$ {
	export class $trip2g_admin_show_usersubgraphaccess extends $.$trip2g_admin_show_usersubgraphaccess {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(/* GraphQL */ `
				query AdminUserSubgraphAccess($id: Int64!) {
					admin {
						userSubgraphAccess(id: $id) {
							expiresAt
						}
					}
				}
			`, { id: this.access_id() })
			return res.admin.userSubgraphAccess!
		}

		expires_at_moment(next?: any) {
			if(next === undefined) return new $mol_time_moment(this.data().expiresAt)
			return next
		}
	}
}
