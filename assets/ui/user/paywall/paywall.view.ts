namespace $.$$ {
	export class $trip2g_user_paywall extends $.$trip2g_user_paywall {
		@$mol_mem
		subgraphs() {
			const sv = this.$.$mol_state_arg.value('subgraph')
			if (sv) {
				return [sv]
			}

			const el = this.dom_node() as HTMLDivElement
			if (!el.dataset.subgraphs) {
				return []
			}

			return this.$.$mol_json_from_string(el.dataset.subgraphs) as string[]
		}

		override message(): string {
			return this.subgraphs().length > 0 ? this.message_offers() : this.message_closed()
		}

		reload_page() {
			window.location.reload()
		}
	}
}
