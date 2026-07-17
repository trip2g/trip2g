namespace $.$$ {
	export class $trip2g_admin_federation_show extends $.$trip2g_admin_federation_show {
		@$mol_mem
		override can_revoke(): boolean {
			return !this.revoked()
		}

		@$mol_mem
		override subgraph_ids( next?: readonly number[] ): readonly number[] {
			if (next === undefined) {
				return []
			}

			const prev = this.subgraph_ids()
			const kid = this.kid()

			const toAdd = next.filter( id => !prev.includes( id ) )
			const toRemove = prev.filter( id => !next.includes( id ) )

			for (const subgraphID of toAdd) {
				const res = $trip2g_admin_federation_show_add({ input: { kid, subgraphID } })
				if (res.admin.data.__typename === 'ErrorPayload') {
					throw new Error(res.admin.data.message)
				}
			}

			for (const subgraphID of toRemove) {
				const res = $trip2g_admin_federation_show_remove({ input: { kid, subgraphID } })
				if (res.admin.data.__typename === 'ErrorPayload') {
					throw new Error(res.admin.data.message)
				}
			}

			return next
		}
	}
}
