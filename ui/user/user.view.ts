namespace $.$$ {
	export class $trip2g_user extends $.$trip2g_user {
		@$mol_mem
		view_name(next?: string): string {
			if (next !== undefined) {
				return this.$.$mol_state_arg.value('view', next) || 'list'
			}

			return next || this.$.$mol_state_arg.value('view') || 'list'
		}

		view_map(): { [key: string]: $.$mol_view } {
			return {
				paywall: this.Paywall(),
				space: this.Space(),
			}
		}

		override component_items(): readonly $mol_view[] {
			return Object.keys(this.view_map()).map(key => this.ComponentButton(key))
		}

		override component_name(key: string): string {
			return key
		}

		override component_click(id: any) {
			this.view_name(id)
		}

		sub() {
			const view = this.view_name()
			return [this.view_map()[view] || this.ComponentList()]
		}
	}
}
