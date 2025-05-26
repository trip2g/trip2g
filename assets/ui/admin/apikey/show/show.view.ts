namespace $.$$ {
	export class $trip2g_admin_apikey_show extends $.$trip2g_admin_apikey_show {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query AdminApiKeyShow($filter: ApiKeyLogsFilterInput!) {
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

			return $trip2g_graphql_make_map( res.admin.apiKeyLogs.nodes.map( ( row, id ) => ( { ...row, id } ) ) )
		}
	}
}