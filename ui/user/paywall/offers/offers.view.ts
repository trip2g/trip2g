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
			return $trip2g_auth_viewer.current().user;
		}

		override buy(id: any) {
			if (this.user()) {
				this.instant_buy(id)
				return
			}
			// if user -> do action
			// show form or auth
			this.$.$mol_state_arg.value('offer', id)
		}

		override instant_buy(id?: any) {
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
						email: this.current_email(),
						offerId: id,
						returnPath: '/secondbrain',
						paymentType: 'CRYPTO' as any,
					},
				}
			)

			if (res.data.__typename === 'ErrorPayload') {
				throw new Error(res.data.message)
			}

			if (res.data.__typename === 'CreatePaymentLinkPayload') {
				window.open(res.data.redirectUrl, '_blank')
				this.$.$mol_state_arg.value('waiting', 'true')
			}
		}

		override to_sign_in() {
			this.$.$mol_state_arg.value('sign_in', 'true')
		}

		@$mol_mem
		current_email(next?: string) {
			if (next !== undefined) {
				this.$.$mol_state_arg.value('email', next)
			}

			return next || this.$.$mol_state_arg.value('email') || '';
		}

		@$mol_mem
		override buy_by_email(email?: string) {
			console.log('buy by email', email)
			if (email) {
				this.current_email(email)
				this.instant_buy()
			}

			return email || '';
		}

		sub() {
			const waiting = this.$.$mol_state_arg.value('waiting')
			if (waiting) {
				return []
			}

			const offer_id = this.$.$mol_state_arg.value('offer')
			if (!offer_id) {
				return [this.List()]
			}

			const user = this.user()
			if (user) {
				const current_subgraphs = this.subgraphs()
				const user_subgraphs = user.subgraphs.map((sub) => sub.name);

				if (user_subgraphs.find((sub) => current_subgraphs.includes(sub))) {
					window.location.reload()
					return [this.Empty()]
				}

				return [this.BuyButton()]
			}

			if (this.$.$mol_state_arg.value('sign_in')) {
				return [this.SignIn()]
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
