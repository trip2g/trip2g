namespace $.$$ {
	export class $trip2g_user_paywall_conversationprompt extends $.$trip2g_user_paywall_conversationprompt {
		override tg_bot_url(): string {
			return this.waitlist().tgBotUrl ?? ''
		}

		override sub() {
			const items: $mol_view[] = [ ...super.sub() ]
			const wl = this.waitlist()

			if (!wl) {
				return []
			}

			if (wl.tgBotUrl) {
				items.push(this.TelegramButton())
			}

			if (wl.emailAllowed) {
				items.push(this.EmailForm())
			}

			return items
		}
	}
}