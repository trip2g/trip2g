namespace $.$$ {
	export class $trip2g_admin_offer_create extends $.$trip2g_admin_offer_create {
		override body() {
			if (this.offer_public_id() !== '') {
				return [this.OfferView()]
			}

			return super.body()
		}

		override submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateOffer($input: CreateOfferInput!) {
						admin {
							data: createOffer(input: $input) {
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
				`,
				{
					input: {
						priceUSD: this.price_usd(),
						subgraphIds: this.subgraph_ids(),
						lifetime: this.lifetime() || null,
						startsAt: this.starts_at_moment()?.toISOString() || null,
						endsAt: this.ends_at_moment()?.toISOString() || null,
					},
				}
			)

			console.log(res)
			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(this.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'CreateOfferPayload') {
				this.offer_public_id(res.admin.data.offer.publicId)
				return
			}

			this.result('Unexpected response type')
		}
	}
}