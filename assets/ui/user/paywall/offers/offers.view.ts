namespace $.$$ {
	const request = ( page_id: number ) => {
		const res = $trip2g_graphql_request(
			`
				query PaywallQuery($filter: ViewerOffersFilter!) {
					viewer {
						offers(filter: $filter) {
							... on ActiveOffers {
								nodes {
									id
									priceUSD
									subgraphs {
										name
									}
								}
							}
							... on SubgraphWaitList {
								tgBotUrl
								emailAllowed
							}
						}
					}
				
				}
			`,
			{
				filter: {
					pageId: page_id,
				}
			}
		)

		return res.viewer.offers
	}

	export type $trip2g_user_paywall_offers_whitelist = Extract<ReturnType<typeof request>, { __typename?: 'SubgraphWaitList' }>

	export class $trip2g_user_paywall_offers extends $.$trip2g_user_paywall_offers {
		page_id(): number {
			const el = document.getElementById('paywall')
			if ( el ) {
				return el.dataset.pageId ? parseInt( el.dataset.pageId, 10 ) : 0
			}

			const page_id = this.$.$mol_state_arg.value( 'page_id' )
			if( page_id ) {
				return parseInt( page_id, 10 )
			}

			throw new Error( 'Page ID not found' )
		}

		@$mol_mem
		data() {
			return request( this.page_id() )
		}

		@$mol_mem
		offers() {
			const data = this.data()
			if( data?.__typename === 'ActiveOffers' ) {
				return $trip2g_graphql_make_map( data.nodes )
			}

			return $trip2g_graphql_make_map( [] as any[] )
		}

		waitlist() {
			const data = this.data()
			if( data?.__typename === 'SubgraphWaitList' ) {
				return data
			}

			return null
		}

		user() {
			return $trip2g_auth_viewer.current().user
		}

		override buy_item( id: any ) {
			if( this.user() ) {
				this.buy( id )
				return
			}
			// if user -> do action
			// show form or auth
			this.$.$mol_state_arg.value( 'offer', id )
		}

		buy( id?: any ) {
			id = id || this.$.$mol_state_arg.value( 'offer' )

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

			if( res.data.__typename === 'ErrorPayload' ) {
				if( res.data.message === 'sign_in_required' ) {
					this.$.$mol_state_arg.value( 'purchase_state', 'sign_in_required' )
					return
				}

				throw new Error( res.data.message )
			}

			if( res.data.__typename === 'CreatePaymentLinkPayload' ) {
				this.$.$mol_state_arg.value( 'purchase_state', 'waiting' )
				this.$.$mol_state_arg.value( 'payment_url', res.data.redirectUrl )

				window.open( res.data.redirectUrl, '_blank' )
			}
		}

		@$mol_mem
		state( next?: 'sign_in_required' | 'sign_in' | 'waiting' | 'init' ) {
			if( next !== undefined ) {
				this.$.$mol_state_arg.value( 'purchase_state', next )
			}

			return next || this.$.$mol_state_arg.value( 'purchase_state' ) || 'init'
		}

		override to_sign_in() {
			this.state( 'sign_in' )
		}

		override reload_me() {
			this.state( 'init' )
			this.$.$mol_state_arg.value( 'purchase_email', null )
			this.$.$mol_state_arg.value( 'purchase_state', null )
			this.$.$mol_state_arg.value( 'offer', null )

			setTimeout( () => window.location.reload(), 50 )
		}

		@$mol_mem
		override current_email( next?: string ) {
			if( next ) {
				this.$.$mol_state_arg.value( 'purchase_email', next || null )
			}

			return next || this.$.$mol_state_arg.value( 'purchase_email' ) || ''
		}

		@$mol_mem
		override buy_by_email( email?: string ) {
			if( email ) {
				this.current_email( email )
				this.buy()
			}

			return email || ''
		}

		override to_payment_page() {
			const url = this.$.$mol_state_arg.value( 'payment_url' )
			if( url ) {
				window.open( url, '_blank' )
			}
		}

		sub() {
			const state = this.state()
			if( state === 'sign_in_required' ) {
				return [ this.EmailExists() ]
			}

			if( state === 'sign_in' ) {
				return [ this.SignIn() ]
			}

			if( state === 'waiting' ) {
				return [ this.Waiting() ]
			}

			const offer_id = this.$.$mol_state_arg.value( 'offer' )
			const user = this.user()
			if( !offer_id || user ) {
				return [ this.List() ]
			}

			return [ this.EnterEmail() ]
		}
	}

	export class $trip2g_user_paywall_offers_enter_email extends $.$trip2g_user_paywall_offers_enter_email {
		override submit() {
			const email = this.email()
			console.log( 'submit', email )
			if( email ) {
				this.handle_email( email )
			}
		}
	}

	export class $trip2g_user_paywall_offers_list extends $.$trip2g_user_paywall_offers_list {
		row( id: any ) {
			return this.offers().get( id )
		}

		override list_items(): readonly $mol_view[] {
			if( this.offers().size() === 0 ) {
				return [ this.Empty() ]
			}

			return this.offers().map( ( key: any ) => this.Item( key ) )
		}

		override row_formatted_price( id: any ): string {
			const val = this.row( id ).priceUSD.toFixed( 2 )
			return `$${ val }`
		}

		override row_description( id: any ): string {
			return this.row( id )
				.subgraphs.map( ( sub: any ) => sub.name )
				.join( ' + ' )
		}
	}
}
