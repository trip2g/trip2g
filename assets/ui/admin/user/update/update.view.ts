namespace $.$$ {
	export class $trip2g_admin_user_update extends $.$trip2g_admin_user_update {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminUserEditQuery($id: Int64!) {
						admin {
							user(id: $id) {
								id
								email
								createdAt
							}
						}
					}
				`,
				{ id: this.user_id() }
			)

			if (!res.admin.user) {
				throw new Error('User not found')
			}

			return res.admin.user
		}

		user_id_string(): string {
			return this.data().id.toString()
		}

		@$mol_mem
		email(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().email || ''
		}

		override email_bid(): string {
			const email = this.email()
			if (!email || email.trim() === '') {
				return 'Email is required'
			}

			// Basic email validation
			const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
			if (!emailRegex.test(email)) {
				return 'Please enter a valid email address'
			}

			return ''
		}

		override submit_allowed(): boolean {
			return this.email_bid() === '' && this.email().trim() !== ''
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminUpdateUser($input: UpdateUserInput!) {
						admin {
							updateUser(input: $input) {
								... on UpdateUserPayload {
									user {
										id
										email
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
						id: this.user_id(),
						email: this.email()
					},
				}
			)

			if (res.admin.updateUser.__typename === 'ErrorPayload') {
				this.result(res.admin.updateUser.message)
				return
			}

			if (res.admin.updateUser.__typename === 'UpdateUserPayload') {
				this.result('User updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}
