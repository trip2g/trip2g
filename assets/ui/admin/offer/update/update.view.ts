namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowOffer($id: Int64!) {
			admin {
				offer(id: $id) {
					id
					publicId
					createdAt
					lifetime
					priceUSD
					startsAt
					endsAt
					subgraphIds
					subgraphs {
						id
						name
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminUpdateOfferMutation($input: UpdateOfferInput!) {
			admin {
				payload: updateOffer(input: $input) {
					... on UpdateOfferPayload {
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

	export class $trip2g_admin_offer_update extends $.$trip2g_admin_offer_update {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.offer_id() })

			if (!res.admin.offer) {
				throw new Error('Offer not found')
			}

			return res.admin.offer
		}

		offer_title(): string {
			return `Offer ${this.data().publicId}`
		}

		offer_public_id(): string {
			return this.data().publicId
		}

		@$mol_mem
		subgraph_ids(next?: number[]): number[] {
			if (next !== undefined) {
				return next
			}

			return this.data().subgraphIds || []
		}

		@$mol_mem
		lifetime(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().lifetime || ''
		}

		@$mol_mem
		price_usd(next?: number): number {
			if (next !== undefined) {
				return next
			}

			return this.data().priceUSD || 0
		}

		@$mol_mem
		starts_at_moment(next?: $mol_time_moment | null): $mol_time_moment | null {
			if (next !== undefined) {
				if (next) {
					next = new $mol_time_moment().merge(next)
				}
				return next
			}

			const startsAt = this.data().startsAt
			return startsAt ? new $mol_time_moment(startsAt) : null
		}

		@$mol_mem
		ends_at_moment(next?: $mol_time_moment | null): $mol_time_moment | null {
			if (next !== undefined) {
				if (next) {
					next = new $mol_time_moment().merge(next)
				}
				return next
			}

			const endsAt = this.data().endsAt
			return endsAt ? new $mol_time_moment(endsAt) : null
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

		submit() {
			const res = mutate({
				input: {
					id: this.offer_id(),
					priceUSD: this.price_usd(),
					subgraphIds: this.subgraph_ids() as number[],
					lifetime: this.lifetime() || null,
					startsAt: $trip2g_moment_toserver(this.starts_at_moment()),
					endsAt: $trip2g_moment_toserver(this.ends_at_moment())
				},
			})

			if (res.admin.payload.__typename === 'ErrorPayload') {
				this.result(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'UpdateOfferPayload') {
				this.result('Offer updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}