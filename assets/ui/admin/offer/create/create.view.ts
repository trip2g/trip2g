namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateOfferMutation($input: CreateOfferInput!) {
			admin {
				payload: createOffer(input: $input) {
					__typename
					... on CreateOfferPayload {
						offer {
							id
							publicId
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_offer_create extends $.$trip2g_admin_offer_create {
		override body() {
			if( this.offer_public_id() !== '' ) {
				return [ this.OfferView() ]
			}

			return super.body()
		}

		override starts_at_moment( next?: $mol_time_moment | null ) {
			if (next) {
				next = new $mol_time_moment().merge(next);
			}

			return super.starts_at_moment(next)
		}

		override ends_at_moment( next?: $mol_time_moment | null ) {
			if (next) {
				next = new $mol_time_moment().merge(next);
			}

			return super.ends_at_moment(next)
		}

		override subgraphs_bid(): string {
			if (this.subgraph_ids().length === 0) {
				return 'Select subgraphs'
			}

			return ''
		}

		override starts_at_bid(): string {
			const startsAt = this.starts_at_moment()
			const endsAt = this.ends_at_moment()
			
			if (startsAt && endsAt && startsAt.valueOf() >= endsAt.valueOf()) {
				return 'Start date must be before end date'
			}

			return ''
		}

		override ends_at_bid(): string {
			const startsAt = this.starts_at_moment()
			const endsAt = this.ends_at_moment()
			
			if (startsAt && endsAt && startsAt.valueOf() >= endsAt.valueOf()) {
				return 'End date must be after start date'
			}

			return ''
		}

		override submit() {
			const res = mutate({
				input: {
					priceUSD: this.price_usd(),
					subgraphIds: this.subgraph_ids() as number[],
					lifetime: this.lifetime() || null,
					startsAt: $trip2g_moment_toserver(this.starts_at_moment()),
					endsAt: $trip2g_moment_toserver(this.ends_at_moment())
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateOfferPayload' ) {
				this.offer_public_id( res.admin.payload.offer.publicId )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}