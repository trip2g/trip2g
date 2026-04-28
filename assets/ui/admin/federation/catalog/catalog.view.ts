namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminListFederationSecrets {
			admin {
				federationSecrets {
					id
					kid
					kbUrl
					description
					createdAt
					createdBy
					revokedAt
					subgraphCount
				}
			}
		}
	`)

	export class $trip2g_admin_federation_catalog extends $.$trip2g_admin_federation_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
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

		override row_subgraph_count( id: any ): string {
			return this.row( id ).subgraphCount.toString()
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
