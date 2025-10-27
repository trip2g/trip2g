namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query Viewer {
			viewer {
				id
				role
				user {
					email
				}
			}
		}
	`)

	export class $trip2g_auth_viewer extends $.$mol_object2 {
		// static method
		@$mol_mem
		static current(reset?: null) {
			return request().viewer
		}
	}
}
