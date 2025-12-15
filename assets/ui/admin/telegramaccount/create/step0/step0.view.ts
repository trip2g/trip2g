namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation StartTelegramAccountAuth($input: AdminStartTelegramAccountAuthInput!) {
			admin {
				payload: startTelegramAccountAuth(input: $input) {
					__typename
					... on AdminStartTelegramAccountAuthPayload {
						authState {
							phone
							state
							passwordHint
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_telegramaccount_create_step0 extends $.$trip2g_admin_telegramaccount_create_step0 {
		override phone_bid(): string {
			if (this.phone().trim() === '') {
				return 'Phone is required'
			}
			return ''
		}

		override api_id_bid(): string {
			if (this.api_id().trim() === '') {
				return 'API ID is required'
			}
			if (!/^\d+$/.test(this.api_id().trim())) {
				return 'API ID must be a number'
			}
			return ''
		}

		override api_hash_bid(): string {
			if (this.api_hash().trim() === '') {
				return 'API Hash is required'
			}
			return ''
		}

		@$mol_mem
		override api_id(next?: string): string {
			return this.$.$mol_state_local.value('telegram_account_api_id', next) || ''
		}

		@$mol_mem
		override api_hash(next?: string): string {
			return this.$.$mol_state_local.value('telegram_account_api_hash', next) || ''
		}

		override submit() {
			const res = mutate({
				input: {
					phone: this.phone().trim(),
					apiId: parseInt(this.api_id().trim(), 10),
					apiHash: this.api_hash().trim()
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'AdminStartTelegramAccountAuthPayload') {
				this.on_success(null)
				return
			}

			this.result('Unexpected response type')
		}
	}
}
