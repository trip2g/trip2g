namespace $.$$ {
	export class $trip2g_admin_list_users extends $.$trip2g_admin_list_users {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(`
				query AdminListUsers {
					admin {
						allUsers {
							nodes {
								id
								email
								createdAt
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allUsers.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key));
		}

		row( id: any ) {
			return this.data().get(id);
		}

		row_id( id: any ): string {
			return this.row(id).id.toString();
		}

		row_email( id: any ): string {
			return this.row(id).email;
		}
	}
}
