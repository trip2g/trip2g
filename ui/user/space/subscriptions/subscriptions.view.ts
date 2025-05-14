namespace $.$$ {
	export class $trip2g_user_space_subscriptions extends $.$trip2g_user_space_subscriptions {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`
				query UserSubscriptions {
					viewer {
						user {
							subgraphAccesses {
								id
								createdAt
								expiresAt
								subgraph {
									name
									homePath
								}
							}
						}
					}
				}
			`)

			if (!res.viewer.user) {
				throw new Error('User not found')
			}

			return $trip2g_graphql_make_map(res.viewer.user.subgraphAccesses)
		}

		override rows() {
			return this.data().map(key => this.Row(key))
		}

		row( id: any ) {
			return this.data().get(id)
		}

		override row_name( id: any ): string {
			return this.row(id).subgraph.name
		}

		override row_uri( id: any ): string {
			return this.row(id).subgraph.homePath
		}

		override row_created_at( id: any ): string {
			return this.row(id).createdAt
		}

		override row_expires_at( id: any ) {
			return this.row(id).expiresAt
		}
	}
}