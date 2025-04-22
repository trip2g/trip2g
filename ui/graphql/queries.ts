namespace $.$$ {

	export const $trip2g_graphql_list_users_query = `query ListUsers {
  admin {
    listUsers {
      nodes {
        id
        email
        createdAt
      }
    }
  }
}`

	export const $trip2g_graphql_list_users_response = $mol_data_record({
		admin: $mol_data_record({
			listUsers: $mol_data_record({
				nodes: $mol_data_array($mol_data_record({
						id: $mol_data_integer,
						email: $mol_data_string,
						createdAt: $mol_data_pipe( $mol_data_string , $mol_time_moment )
					}))
			})
		})
	})

	export const $trip2g_graphql_list_users = () =>
		$trip2g_graphql_list_users_response($trip2g_graphql_request($trip2g_graphql_list_users_query))

}