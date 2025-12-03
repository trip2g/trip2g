namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation CompleteTelegramAccountAuth($input: AdminCompleteTelegramAccountAuthInput!) {
			admin {
				payload: completeTelegramAccountAuth(input: $input) {
					... on AdminCompleteTelegramAccountAuthPayload {
						account {
							id
							phone
							displayName
							isPremium
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramaccount_create_step1 extends $.$trip2g_admin_telegramaccount_create_step1 {
		phone(): string {
			return this.$.$mol_state_arg.value('phone') || ''
		}

		override code_bid(): string {
			if (this.code().trim() === '') {
				return 'Code is required'
			}
			return ''
		}

		override submit() {
			const input: any = {
				phone: this.phone(),
				code: this.code().trim()
			}

			const pwd = this.password().trim()
			if (pwd !== '') {
				input.password = pwd
			}

			const res = mutate({ input })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'AdminCompleteTelegramAccountAuthPayload') {
				this.account(res.admin.payload.account)
				this.result('Account created successfully!')
				return
			}

			this.result('Unexpected response type')
		}
	}
}
