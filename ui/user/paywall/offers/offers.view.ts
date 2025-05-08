namespace $.$$ {
	export class $trip2g_user_paywall_offers extends $.$trip2g_user_paywall_offers {
		@$mol_mem
		data() {
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
					subgraphs: this.subgraphs() as string[],
				}
			)

			return $trip2g_graphql_make_map(res.viewer.offers)
		}

		row(id: any) {
			return this.data().get(id)
		}

		override list_items(): readonly $mol_view[] {
			return this.data().map(key => this.Item(key))
		}

		override row_formatted_price(id: any): string {
			return this.row(id).priceUSD.toFixed(2)
		}
	}
}
