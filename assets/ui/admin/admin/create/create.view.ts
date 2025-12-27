namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminCreateAdmin($input: CreateAdminInput!) {
			admin {
				data: createAdmin(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on CreateAdminPayload {
						admin {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_admin_create extends $.$trip2g_admin_admin_create {
		override user_id(): number {
			const id = this.$.$mol_state_arg.value('user_id')
			return id ? parseInt(id, 10) : 0
		}

		override user_id_string(): string {
			return this.user_id().toString()
		}

		override submit() {
			const userId = this.user_id()
			if (!userId) {
				throw new Error('User ID is required')
			}

			const res = mutate({
				input: {
					userId: userId,
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'CreateAdminPayload') {
				this.result('Admin created successfully')
				this.$.$mol_state_arg.value('id', res.admin.data.admin.id.toString())
				return
			}

			throw new Error('Unexpected response type')
		}
	}
}
