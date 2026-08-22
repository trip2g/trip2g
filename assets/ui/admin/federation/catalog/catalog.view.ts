namespace $.$$ {
	export class $trip2g_admin_federation_catalog extends $.$trip2g_admin_federation_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_federation_catalog_list()
			return $trip2g_graphql_make_map( res.admin.federationSecrets )
		}

		@$mol_mem
		spreads(): any {
			return {
				add_inbound: this.AddInboundForm(),
				add_outbound: this.AddOutboundForm(),
				...this.data().mapKeys( key => this.Content( key ) )
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add_inbound' && id !== 'add_outbound' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_kid( id: any ): string {
			return this.row( id ).kid
		}

		override row_kb_url( id: any ): string {
			return this.row( id ).kbUrl ?? '—'
		}

		// kb_url is the direction: a null one is a key a peer signs with to call
		// here, a filled one is a key this instance signs with to call out. The
		// column exists because that is the first thing an operator needs to know
		// about a row and a dash in the URL column does not say it.
		override row_direction( id: any ): string {
			return this.row( id ).kbUrl ? 'outbound' : 'inbound'
		}

		// Empty means the pairing still holds the key it was created with — for an
		// outbound secret, the one that travelled out of band.
		override row_rotated_at( id: any ): string {
			return this.row( id ).rotatedAt ?? ''
		}

		override row_subgraph_ids( id: any ): readonly number[] {
			return this.row( id ).subgraphIds.map( ( value: any ) => Number( value ) )
		}

		// Two different numbers under one heading, because the question differs by
		// direction. An inbound key carries scope this instance granted, and that
		// is a local count. An outbound key carries none of its own: what it may
		// read was decided on the peer, so the only truthful number comes from
		// asking the peer. Showing the local count there would report what an
		// inbound key with the same kid happens to grant, which is backwards.
		@$mol_mem_key
		override row_subgraph_count( id: any ): string {
			const row = this.row( id )
			if ( !row.kbUrl ) return row.subgraphCount.toString()

			const res = $trip2g_admin_federation_show_scope({ kid: row.kid })
			if ( res.admin.data.__typename === 'ErrorPayload' ) return '?'

			return res.admin.data.subgraphs.length.toString()
		}

		override row_created_at( id: any ): string {
			const m = new $mol_time_moment( this.row( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override row_revoked_at( id: any ): string {
			return this.row( id ).revokedAt ?? ''
		}

		override row_revoked( id: any ): boolean {
			return !!this.row( id ).revokedAt
		}
	}
}
