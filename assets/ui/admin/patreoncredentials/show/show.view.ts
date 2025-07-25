namespace $.$$ {
	export class $trip2g_admin_patreoncredentials_show extends $.$trip2g_admin_patreoncredentials_show {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminPatreonCredentialsById($id: Int64!) {
					admin {
						patreonCredentials(id: $id) {
							createdAt
							creatorAccessToken

							createdBy {
								email
							}

							tiers {
								nodes {
									id
									missedAt
									title
									amountCents
								}
							}
							
							members {
								nodes {
									email
									status
									currentTier {
										title
									}
								}
							}
						}
					}
				}
			`, {
				id: this.credentials_id(),
			})

			return res.admin.patreonCredentials
		}

		@$mol_mem
		override tools() {
			const data = this.data()
			if( !data ) return []

			const tools = []
			
			if( data.state === 'ACTIVE' ) {
				tools.push( this.DeleteButton() )
			}
			
			if( data.state === 'DELETED' ) {
				tools.push( this.RestoreButton() )
			}

			return tools
		}

		override credentials_id_string(): string {
			return this.credentials_id().toString()
		}

		override credentials_state(): string {
			const data = this.data()
			if( !data ) return '-'
			return data.state === 'ACTIVE' ? 'Active' : 'Deleted'
		}

		override credentials_token(): string {
			const data = this.data()
			if( !data ) return '-'
			return data.creatorAccessToken
		}

		override credentials_created_at(): string {
			const data = this.data()
			if( !data ) return '-'
			const m = new $mol_time_moment( data.createdAt )
			return m.toString( 'YYYY-MM-DD hh:mm' )
		}

		override credentials_created_by(): string {
			const data = this.data()
			if( !data ) return '-'
			return data.createdBy.email || '-'
		}
	}
}