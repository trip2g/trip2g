namespace $.$$ {
	export class $trip2g_user_paywall_activepurchases extends $.$trip2g_user_paywall_activepurchases {
		subgraphs(): string[] {
			throw new Error('Not implemented')
		}

		@$mol_mem
		data(next?: null) {
			const subgraphs = this.subgraphs()
			if (subgraphs.length === 0) {
				return $trip2g_graphql_make_map([])
			}

			const res = $trip2g_graphql_request(
				`
				query PaywallActivePurchaseQuery($input: ActivePurchasesInput!) {
					viewer {
						activePurchases(input: $input) {
							id
							status
							successful
						}
					}
				}
			`,
				{
					input: {
						email: this.current_email() || null,
						subgraphs,
					},
				}
			)

			const done = this.$.$mol_state_arg.value('done') === 'true'

			if (!done) {
				setTimeout(() => this.data(null), 3000)
			}

			const rows = res.viewer.activePurchases

			// avoid infinite reload if something went wrong
			if (rows.every(row => row.successful) && !done) {
				this.$.$mol_state_arg.value('done', 'true')

				// wait for the state to be set
				setTimeout(() => {
					window.location.reload()
				}, 10)
			}

			return $trip2g_graphql_make_map(rows)
		}

		override list_items(): readonly $mol_view[] {
			return this.data().map(key => this.Item(key))
		}

		purchase(id: any) {
			return this.data().get(id)
		}

		override row_status(id: any) {
			return this.purchase(id).status
		}
	}
}
