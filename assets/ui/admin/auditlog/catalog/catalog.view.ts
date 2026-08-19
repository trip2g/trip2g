namespace $.$$ {
	export class $trip2g_admin_auditlog_catalog extends $.$trip2g_admin_auditlog_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_auditlog_catalog_list({
				filter: {},
			})

			return $trip2g_graphql_make_map( res.admin.auditLogs.nodes )
		}

		override rows() {
			return this.data().map( id => this.Row( id ) )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		row_id( id: any ): string {
			return this.row( id ).id.toString()
		}

		row_id_number( id: any ): number {
			return this.row( id ).id
		}

		// $trip2g_admin_labeler_moment formats the value itself; formatting here
		// too fed it a rendered date it could not parse back.
		row_created_at( id: any ): string {
			return this.row( id ).createdAt
		}

		row_level( id: any ): string {
			return this.row( id ).level
		}

		row_message( id: any ): string {
			return this.row( id ).message
		}

		row_params( id: any ): string {
			const params = this.row( id ).params
			try {
				const parsed = JSON.parse( params )
				// Return a compact string representation
				return JSON.stringify( parsed )
			} catch {
				return params || '-'
			}
		}
	}
}
