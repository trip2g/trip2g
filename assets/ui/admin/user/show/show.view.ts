namespace $.$$ {
	export class $trip2g_admin_user_show extends $.$trip2g_admin_user_show {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(`
				query AdminUserShow($id: Int64!) {
					admin {
						user(id: $id) {
							id
							email
							createdAt
						}
					}
				}
			`, {
				id: this.user_id()
			})

			return res.admin.user
		}

		override user_id_string(): string {
			return this.data().id.toString()
		}

		override user_email(): string {
			return this.data().email || '-'
		}

		override user_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm:ss' )
		}
	}
}