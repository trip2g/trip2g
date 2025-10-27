namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query PaywallActivePurchaseQuery {
			viewer {
				activePurchases {
					id
					status
					successful
				}
			}
		}
	`)

	export class $trip2g_user_paywall_activepurchases extends $.$trip2g_user_paywall_activepurchases {
		subgraphs(): string[] {
			throw new Error('Not implemented')
		}

		@$mol_mem
		update_marker(next?: null) {
			return next;
		}

		@$mol_mem
		data(next?: null) {
			this.update_marker()

			const subgraphs = this.subgraphs()
			if (subgraphs.length === 0) {
				return $trip2g_graphql_make_map([])
			}

			const res = request()

			const done = this.$.$mol_state_arg.value('done') === 'true'

			let updateInterval: NodeJS.Timeout | null = null

			if (!done) {
				updateInterval = setTimeout(() => this.update_marker(null), 3000)
			}

			const rows = res.viewer.activePurchases

			// avoid infinite reload if something went wrong
			if (rows.every(row => row.successful) && !done && rows.length > 0) {
				// set the state to done
				this.$.$mol_state_arg.value('done', 'true')

				// wait for the state to be set
				setTimeout(() => {
					window.location.reload()
				}, 10)
			}

			const map = $trip2g_graphql_make_map(rows)

			return Object.assign(map, {
				destructor: () => {
					if (updateInterval) {
						clearTimeout(updateInterval)
					}
				},
			})
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
