namespace $.$$ {
	export class $trip2g_admin_list_users extends $.$trip2g_admin_list_users {
		@$mol_mem
		data() {
			const res = $trip2g_graphql_request(/* GraphQL */ `
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

			const map: { [ id: number ]: any } = {};

			res.admin.allUsers.nodes.forEach( ( row ) => {
				map[ row.id ] = row
			})

			return {
				map,
				ids: Object.keys( map ),
			}
		}

		@$mol_mem
		spreads(): any {
			const pages: { [ id: string ]: any } = {};

			this.data().ids.forEach( (id) => {
				pages[id] = this.Content(id);
			});

			return pages;
		}

		row_id( id: any ): string {
			return this.data().map[ id ].id.toString();
		}

		row_email( id: any ): string {
			return this.data().map[ id ].email;
		}
	}
}
