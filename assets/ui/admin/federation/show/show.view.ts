namespace $.$$ {
	const mutateAdd = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminAddFederationSecretSubgraph($input: AddFederationSecretSubgraphInput!) {
			admin {
				data: addFederationSecretSubgraph(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on AddFederationSecretSubgraphPayload {
						success
					}
				}
			}
		}
	`)

	const mutateRemove = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminRemoveFederationSecretSubgraph($input: RemoveFederationSecretSubgraphInput!) {
			admin {
				data: removeFederationSecretSubgraph(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on RemoveFederationSecretSubgraphPayload {
						success
					}
				}
			}
		}
	`)

	export class $trip2g_admin_federation_show extends $.$trip2g_admin_federation_show {
		@$mol_mem
		override can_revoke(): boolean {
			return !this.revoked()
		}

		@$mol_mem
		override subgraph_ids( next?: number[] ): number[] {
			if (next === undefined) {
				return this.$.$mol_mem_cached( () => [] as number[] )
			}

			const prev = this.subgraph_ids()
			const kid = this.kid()

			const toAdd = next.filter( id => !prev.includes( id ) )
			const toRemove = prev.filter( id => !next.includes( id ) )

			for (const subgraphID of toAdd) {
				const res = mutateAdd({ input: { kid, subgraphID } })
				if (res.admin.data.__typename === 'ErrorPayload') {
					throw new Error(res.admin.data.message)
				}
			}

			for (const subgraphID of toRemove) {
				const res = mutateRemove({ input: { kid, subgraphID } })
				if (res.admin.data.__typename === 'ErrorPayload') {
					throw new Error(res.admin.data.message)
				}
			}

			return next
		}
	}
}
