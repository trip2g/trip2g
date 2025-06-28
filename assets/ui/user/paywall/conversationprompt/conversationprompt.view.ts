namespace $.$$ {
	export class $trip2g_user_paywall_conversationprompt extends $.$trip2g_user_paywall_conversationprompt {
		override tg_bot_url(): string {
			return this.waitlist().tgBotUrl ?? ''
		}

		override sub() {
			const items: $mol_view[] = [ ...super.sub() ]

			if (this.waitlist().tgBotUrl) {
				items.push(this.TelegramButton())
			}

			if (this.waitlist().emailAllowed) {
				items.push(this.EmailForm())
			}

			return items
		}
	}
}