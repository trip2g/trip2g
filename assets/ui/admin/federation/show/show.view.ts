namespace $.$$ {
	export class $trip2g_admin_federation_show extends $.$trip2g_admin_federation_show {
		// Scope is what a BASE grants a peer, so it belongs to an inbound key only.
		// An outbound one is this instance asking, and what it may read is decided
		// on the other side entirely. Offering the editor there would be worse than
		// useless: the scope table keys on kid, so a pairing wired in both
		// directions under one kid would have the outbound card silently editing
		// what the inbound key surfaces.
		@$mol_mem
		override info_sub(): readonly any[] {
			const scope = this.direction() === 'inbound'
				? this.SubgraphsSection_labeler()
				: this.OutboundScopeNote()

			return [...super.info_sub(), scope]
		}

		// Asked of the peer, because only the peer knows: scope is granted on its
		// side and nothing about it is recorded here. Reporting a local guess would
		// be worse than reporting nothing — a pairing scoped to nothing answers
		// every search with an empty result, which is indistinguishable from a
		// query that matched nothing unless someone says which it is.
		@$mol_mem
		peer_scope(): string {
			if (this.direction() !== 'outbound') return ''

			const res = $trip2g_admin_federation_show_scope({ kid: this.kid() })

			if (res.admin.data.__typename === 'ErrorPayload') {
				return res.admin.data.message
			}

			const subgraphs = res.admin.data.subgraphs
			if (subgraphs.length === 0) {
				return 'nothing — the peer authenticates this key but grants it no subgraph, so every search through it comes back empty'
			}

			return subgraphs.join(', ')
		}

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

		// Rotation is this instance asking a peer to adopt a new key, so it exists
		// only where there is a peer to ask. An inbound row is the other side's
		// credential for calling here; rotating it is their move, not ours.
		@$mol_mem
		can_rotate(): boolean {
			return this.direction() === 'outbound' && !this.revoked()
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
