namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminUserShow($id: Int64!) {
			admin {
				user(id: $id) {
					id
					email
					createdAt
				}
			}
		}
	`)

	export class $trip2g_admin_user_show extends $.$trip2g_admin_user_show {
		action() {
			return this.$.$mol_state_arg.value('action') || 'view';
		}

		override body() {
			if (this.action() === 'update') {
				return [this.UpdateForm()]
			}

			return super.body()
		}
		@$mol_mem
		data() {
			const res = request({
				id: this.user_id()
			})

			if (!res.admin.user) {
				throw new Error('User not found')
			}

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
