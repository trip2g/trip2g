namespace $.$$ {
	const request = (subgraphs: string[]) => {
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
				subgraphs,
			}
		)

		return res.viewer
	}

	export class $trip2g_user_paywall_offers extends $.$trip2g_user_paywall_offers {
		@$mol_mem
		data() {
			return request(this.subgraphs() as string[])
		}

		@$mol_mem
		offers() {
			return $trip2g_graphql_make_map(this.data().offers)
		}

		user() {
			return $trip2g_auth_viewer.current().user
		}

		override buy_item(id: any) {
			if (this.user()) {
				this.buy(id)
				return
			}
			// if user -> do action
			// show form or auth
			this.$.$mol_state_arg.value('offer', id)
		}

		buy(id?: any) {
			id = id || this.$.$mol_state_arg.value('offer')

			const res = $trip2g_graphql_request(
				`
				mutation CreatePaymentLink($input: CreatePaymentLinkInput!) {
					data: createPaymentLink(input: $input) {
						... on CreatePaymentLinkPayload {
							redirectUrl
						}
						... on ErrorPayload {
							message
						}
					}
				}
				`,
				{
					input: {
						email: this.current_email() || null,
						offerId: id,
						returnPath: '/secondbrain',
						paymentType: 'CRYPTO' as any,
					},
				}
			)

			if (res.data.__typename === 'ErrorPayload') {
				if (res.data.message === 'sign_in_required') {
					this.$.$mol_state_arg.value('purchase_state', 'sign_in_required')
					return
				}

				throw new Error(res.data.message)
			}

			if (res.data.__typename === 'CreatePaymentLinkPayload') {
				this.$.$mol_state_arg.value('purchase_state', 'waiting')

				const s = window.open(res.data.redirectUrl, '_blank')
				if (s === null) {
					this.$.$mol_state_arg.value('payment_url', res.data.redirectUrl)
				}
			}
		}

		@$mol_mem
		state(next?: 'sign_in_required' | 'sign_in' | 'waiting' | 'init') {
			if (next !== undefined) {
				this.$.$mol_state_arg.value('purchase_state', next)
			}

			return next || this.$.$mol_state_arg.value('purchase_state') || 'init'
		}

		override to_sign_in() {
			this.state('sign_in')
		}

		override reload_me() {
			this.state('init')
			this.$.$mol_state_arg.value('purchase_email', null)
			this.$.$mol_state_arg.value('purchase_state', null)
			this.$.$mol_state_arg.value('offer', null)

			setTimeout(() => window.location.reload(), 50)
		}

		@$mol_mem
		override current_email(next?: string) {
			console.log('call current_email', next)
			if (next) {
				this.$.$mol_state_arg.value('purchase_email', next || null)
			}

			return next || this.$.$mol_state_arg.value('purchase_email') || ''
		}

		@$mol_mem
		override buy_by_email(email?: string) {
			console.log('buy_by_email', email)
			if (email) {
				this.current_email(email)
				this.buy()
			}

			return email || ''
		}

		sub() {
			const state = this.state()
			if (state === 'sign_in_required') {
				return [this.EmailExists()]
			}

			if (state === 'sign_in') {
				return [this.SignIn()]
			}

			if (state === 'waiting') {
				return [this.Waiting()]
			}

			const offer_id = this.$.$mol_state_arg.value('offer')
			const user = this.user()
			if (!offer_id || user) {
				return [this.List()]
			}

			return [this.EnterEmail()]
		}
	}

	export class $trip2g_user_paywall_offers_enter_email extends $.$trip2g_user_paywall_offers_enter_email {
		override submit() {
			const email = this.email()
			console.log('submit', email)
			if (email) {
				this.handle_email(email)
			}
		}
	}

	export class $trip2g_user_paywall_offers_list extends $.$trip2g_user_paywall_offers_list {
		row(id: any) {
			return this.offers().get(id)
		}

		override list_items(): readonly $mol_view[] {
			return this.offers().map((key: any) => this.Item(key))
		}

		override row_formatted_price(id: any): string {
			const val = this.row(id).priceUSD.toFixed(2)
			return `$${val}`
		}

		override row_description(id: any): string {
			return this.row(id)
				.subgraphs.map((sub: any) => sub.name)
				.join(' + ')
		}
	}
}
