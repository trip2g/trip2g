namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminApiKeyShowQuery($filter: ApiKeyLogsFilterInput!) {
			admin {
				apiKeyLogs(filter: $filter) {
					nodes {
						createdAt
						actionName
						ip
					}
				}
			}
		}
	`)

	export class $trip2g_admin_apikey_show extends $.$trip2g_admin_apikey_show {
		@$mol_mem
		data( reset?: null ) {
			const res = request({
				filter: {
					apiKeyId: this.apikey_id(),
				},
			})

			return $trip2g_graphql_make_map( res.admin.apiKeyLogs.nodes.map( ( row, id ) => ( { ...row, id } ) ) )
		}

		override body() {
			if (this.disabled()) {
				return super.body()
			}

			return [this.DisableButton(), ...super.body()]
		}

		override logs() {
			return this.data().map((_, idx) => this.LogRow(idx))
		}

		log(idx: any) {
			return this.data().get(idx.toString())
		}

		override log_created_at( id: any ) {
			const m = new $mol_time_moment( this.log( id ).createdAt )
			return m.toString( 'YYYY-MM-DD' )
		}

		override log_action_name( id: any ): string {
			return this.log( id ).actionName
		}

		override log_ip( id: any ): string {
			return this.log( id ).ip
		}
	}
}