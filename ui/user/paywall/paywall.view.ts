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

		offer(id: any) {
			return this.offers().get(id)
		}

		override list_items(): readonly $mol_view[] {
			return this.offers().map(key => this.Item(key))
		}

		override row_formatted_price(id: any): string {
			return this.offer(id).priceUSD.toFixed(2)
		}
	}
}
