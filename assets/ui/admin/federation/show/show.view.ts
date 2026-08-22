namespace $.$$ {
	export class $trip2g_admin_federation_show extends $.$trip2g_admin_federation_show {
		// Said in words rather than left blank: an empty cell reads as "unknown",
		// and the difference between "never rotated" and "rotated at" is the whole
		// question of whether the key that was handed over is still the live one.
		@$mol_mem
		rotated_label(): string {
			const at = this.rotated_at()
			return at ? new $mol_time_moment( at ).toString( 'YYYY-MM-DD hh:mm' ) : 'never — still the key it was created with'
		}

		@$mol_mem
		override can_revoke(): boolean {
			return !this.revoked()
		}

		// The editor has to start from what is actually granted. Starting from an
		// empty list showed every key as having no access however many it had, and
		// made removal impossible: the diff below can only take away what it was
		// shown in the first place.
		@$mol_mem
		override subgraph_ids( next?: readonly number[] ): readonly number[] {
			if (next === undefined) {
				return this.granted_subgraph_ids()
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
