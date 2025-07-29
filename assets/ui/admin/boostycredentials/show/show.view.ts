namespace $.$$ {
	const request = ( id: number ) => {
		const res = $trip2g_graphql_request( `
				query AdminBoostyCredentialsById($id: Int64!) {
					admin {
						boostyCredentials(id: $id) {
							createdAt
							deviceId
							blogName
							state

							createdBy {
								email
							}

							tiers {
								nodes {
									id
									name
								}
							}
							
							members {
								nodes {
									email
									status
								}
							}
						}
					}
				}
			`, { id } )

		if( !res.admin.boostyCredentials ) {
			throw new Error( `Boosty credentials with ID ${id} not found` )
		}

		return res.admin.boostyCredentials
	}

	type Tier = ReturnType<typeof request>[ 'tiers' ][ 'nodes' ][ 0 ]
	type Member = ReturnType<typeof request>[ 'members' ][ 'nodes' ][ 0 ]

	export class $trip2g_admin_boostycredentials_show extends $.$trip2g_admin_boostycredentials_show {
		@$mol_mem
		data( reset?: null ) {
			return request( this.credentials_id() )
		}

		@$mol_mem
		override tools() {
			const data = this.data()

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
			return this.data().state
		}

		override credentials_device_id(): string {
			return this.data().deviceId
		}

		override credentials_blog_name(): string {
			return this.data().blogName
		}

		override credentials_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD hh:mm' )
		}

		override credentials_created_by(): string {
			return this.data().createdBy.email || '-'
		}

		override tiers() {
			return this.data().tiers.nodes
		}

		override members() {
			return this.data().members.nodes
		}
	}

	export class $trip2g_admin_boostycredentials_show_tiers extends $.$trip2g_admin_boostycredentials_show_tiers {
		override items() {
			const rows = this.data_rows() as Tier[]
			return rows.map( ( _, idx ) => this.Row( idx ) )
		}

		@$mol_mem
		row( id: any ): Tier {
			return this.data_rows()[ id ]
		}

		override row_id_string( id: any ) {
			return this.row( id ).id.toString()
		}

		override row_title( id: any ) {
			return this.row( id ).name
		}
	}

	export class $trip2g_admin_boostycredentials_show_members extends $.$trip2g_admin_boostycredentials_show_members {
		override items() {
			const rows = this.data_rows() as Member[]
			return rows.map( ( _, idx ) => this.Row( idx ) )
		}

		@$mol_mem
		row( id: any ): Member {
			return this.data_rows()[ id ]
		}

		row_email( id: any ) {
			return this.row( id ).email || '-'
		}

		row_status( id: any ) {
			return this.row( id ).status || '-'
		}

		row_current_tier( id: any ) {
			return '-'
		}
	}
}