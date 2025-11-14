namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminHealthChecks {
			admin {
				healthChecks {
					id
					status
					description
				}
			}
		}
	`)

	export class $trip2g_admin_healthchecks extends $.$trip2g_admin_healthchecks {
		@$mol_mem
		data() {
			const res = request()
			return $trip2g_graphql_make_map(res.admin.healthChecks);
		}

		@$mol_mem
		override rows() {
			return this.data().map( id => this.Row( id ) )
		}

		@$mol_mem
		row(id: any) {
			const row = this.data().get(id)
			if (!row) throw new Error('HealthCheck not found')
			return row
		}

		override row_id( id: any ) {
			return this.row(id).id
		}

		override row_status( id: any ) {
			return this.row(id).status
		}

		override row_description( id: any ) {
			return this.row(id).description
		}

		override row_status_color( id: any ) {
			if (this.row_status( id ) !== 'OK') {
				return 'red'
			}

			return super.row_status_color( id )
		}

		override row_status_text_color( id: any ) {
			if (this.row_status( id ) !== 'OK') {
				return 'white'
			}

			return super.row_status_text_color( id )
		}
	}
}
