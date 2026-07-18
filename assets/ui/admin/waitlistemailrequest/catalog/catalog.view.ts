namespace $.$$ {
	export class $trip2g_admin_waitlistemailrequest_catalog extends $.$trip2g_admin_waitlistemailrequest_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_waitlistemailrequest_catalog_list()

			// Use email as unique identifier since it's the primary key
			return new Map( res.admin.allWaitListEmailRequests.nodes.map( node => [node.email, node] as const ) )
		}

		override rows() {
			return Array.from( this.data().keys() ).map( email => this.Row( email ) )
		}

		row( email: any ) {
			const row = this.data().get( email )
			if( !row ) throw new Error( 'WaitListEmailRequest not found' )
			return row
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