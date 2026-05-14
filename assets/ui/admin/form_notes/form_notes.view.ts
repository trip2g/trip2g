namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminFormNotes {
			admin {
				formNotes {
					id
					path
					submitCount
					lastSubmitAt
				}
			}
		}
	`)

	export class $trip2g_admin_form_notes extends $.$trip2g_admin_form_notes {
		@$mol_mem
		data( reset?: null ) {
			const res = request()
			return res.admin.formNotes as any[]
		}

		override rows() {
			return this.data().map( ( _: any, i: number ) => this.Row( i ) )
		}

		row( i: number ) {
			return this.data()[ i ]
		}

		override row_path( i: number ): string {
			return this.row( i ).path
		}

		override row_count( i: number ): string {
			return String( this.row( i ).submitCount )
		}

		override row_last( i: number ): string {
			const ts = this.row( i ).lastSubmitAt
			if( !ts ) return '-'
			return new $mol_time_moment( ts ).toString( 'DD.MM.YYYY hh:mm:ss' )
		}
	}
}
