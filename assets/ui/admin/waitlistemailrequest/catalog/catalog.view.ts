namespace $.$$ {
	export class $trip2g_admin_waitlistemailrequest_catalog extends $.$trip2g_admin_waitlistemailrequest_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminWaitListEmailRequests {
					admin {
						allWaitListEmailRequests {
							nodes {
								email
								createdAt
								ip
								notePath
							}
						}
					}
				}
			`)

			// Use email as unique identifier since it's the primary key
			return new Map( res.admin.allWaitListEmailRequests.nodes.map( (node: any) => [node.email, node] ) )
		}

		override rows() {
			return Array.from( this.data().keys() ).map( email => this.Row( email ) )
		}

		row( email: any ) {
			return this.data().get( email )
		}

		row_email( email: any ): string {
			return this.row( email ).email
		}

		row_created_at( email: any ): string {
			const timestamp = this.row( email ).createdAt
			return new $mol_time_moment( timestamp ).toString( 'DD.MM.YYYY hh:mm:ss' )
		}

		row_ip( email: any ): string {
			return this.row( email ).ip || '-'
		}

		row_note_path( email: any ): string {
			return this.row( email ).notePath
		}
	}
}