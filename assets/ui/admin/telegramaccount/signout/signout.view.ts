namespace $.$$ {
	export class $trip2g_admin_telegramaccount_signout extends $.$trip2g_admin_telegramaccount_signout {
		signout() {
			const res = $trip2g_admin_telegramaccount_signout_signout({
				input: {
					id: String(this.account_id()),
				},
			})

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message)
			}

			if (res.admin.data.__typename === 'AdminSignOutTelegramAccountPayload') {
				this.$.$mol_state_arg.value('id', null)
				this.$.$mol_state_arg.value('action', null)
			}
		}
	}
}
