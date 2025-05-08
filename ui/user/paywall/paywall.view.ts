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

		@$mol_mem
		offers() {
			const res = $trip2g_graphql_request(
				`
				query PaywallQuery($subgraphs: [String!]!) {
					viewer {
						offers(subgraphs: $subgraphs) {
							id
							priceUSD
							subgraphs {
								name
							}
						}
					}
				
				}
			`,
				{
					subgraphs: this.subgraphs(),
				}
			)

			return $trip2g_graphql_make_map(res.viewer.offers)
		}

		@$mol_mem
		active_purchases(next?: null) {
			const res = $trip2g_graphql_request(`
				query PaywallActivePurchaseQuery($input: ActivePurchasesInput!) {
					viewer {
						activePurchases(input: $input) {
							id
							status
							successful
						}
					}
				}
			`, {
				input: {
					email: 'hello@example.com',
					subgraphs: this.subgraphs(),
				},
			})

			setTimeout(() => {
				this.active_purchases(null)
			}, 3000);

			const rows = res.viewer.activePurchases;

			if (rows.every(row => row.successful)) {
				const done = this.$.$mol_state_arg.value('done');

				// avoid infinite reload if something went wrong
				if (done !== 'true') {
					this.$.$mol_state_arg.value('done', 'true')
					setTimeout(() => {
						window.location.reload();
					}, 10);
				}
			}

			return $trip2g_graphql_make_map(rows);
		}

		offer(id: any) {
			return this.offers().get(id)
		}

		activePurchase(id: any) {
			return this.active_purchases().get(id)
		}

		override active_purchase_items(): readonly ( $mol_view )[] {
			return this.active_purchases().map(key => this.ActivePurchase(key))
		}

		override active_purchase_status( id: any ): string {
			return this.activePurchase(id).status
		}

		override list_items(): readonly $mol_view[] {
			return this.offers().map(key => this.Item(key))
		}

		override row_formatted_price(id: any): string {
			return this.offer(id).priceUSD.toFixed(2)
		}
	}
}
