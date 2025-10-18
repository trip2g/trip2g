namespace $.$$ {
	export class $trip2g_admin_user_create extends $.$trip2g_admin_user_create {
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

		override submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateUser($input: CreateUserInput!) {
						admin {
							createUser(input: $input) {
								... on CreateUserPayload {
									user {
										id
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
						email: this.email()
					},
				}
			)

			if( res.admin.createUser.__typename === 'ErrorPayload' ) {
				this.result( res.admin.createUser.message )
				return
			}

			if( res.admin.createUser.__typename === 'CreateUserPayload' ) {
				this.$.$mol_state_arg.value( 'id', res.admin.createUser.user.id.toString() )
				this.result( 'User created successfully' )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
