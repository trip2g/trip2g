namespace $.$$ {
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
			const password = this.password().trim()

			const res = $trip2g_admin_telegramaccount_create_step1_complete({
				input: {
					phone: this.phone(),
					code: this.code().trim(),
					password: password !== '' ? password : undefined,
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'AdminCompleteTelegramAccountAuthPayload') {
				this.on_success(res.admin.payload.account.id)
				this.result('Account created successfully!')
				return
			}

			this.result('Unexpected response type')
		}
	}
}
