namespace $.$$ {
	export class $trip2g_admin_telegramaccount_show_dialogs_import extends $.$trip2g_admin_telegramaccount_show_dialogs_import {
		account_id() {
			return this.$.$mol_state_arg.value('account_id') || ''
		}

		chat_id() {
			return this.$.$mol_state_arg.value('chat_id') || ''
		}

		default_base_path() {
			return this.$.$mol_state_arg.value('default_base_path') || ''
		}

		@$mol_mem
		override base_path(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.default_base_path()
		}

		submit(event?: Event) {
			const basePath = this.base_path().trim()
			if (!basePath) {
				throw new Error('Base path is required')
			}

			const res = $trip2g_admin_telegramaccount_show_dialogs_import_channel({
				input: {
					accountId: this.account_id(),
					channelId: this.chat_id(),
					basePath: basePath,
					skipExists: this.skip_exists(),
					withMedia: this.with_media(),
				},
			})

			const { payload } = res.admin

			if (payload.__typename === 'ErrorPayload') {
				throw new Error(payload.message)
			}

			this.submit_title('Import started')
		}
	}
}
