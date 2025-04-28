namespace $.$$ {
	export class $trip2g_admin_list_users extends $.$trip2g_admin_list_users {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`
				query AdminListUsers {
					admin {
						allUsers {
							nodes {
								id
								email
								createdAt
								ban { reason }
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allUsers.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key))
		}

		@$mol_mem
		pages() {
			const ban_id = this.$.$mol_state_arg.value('ban_id')
			if (ban_id) {
				return [this.Menu(), this.UserBan(ban_id)]
			}

			return super.pages()
		}

		ban_button(id: any): $mol_view {
			if (this.row(id).ban) {
				return this.UserUnbanButton(id)
			}

			return this.UserBanButton(id)
		}

		row(id: any) {
			return this.data().get(id)
		}

		row_id_number(id: any): number {
			return id
		}

		row_id(id: any): string {
			return this.row(id).id.toString()
		}

		row_email(id: any): string {
			return this.row(id).email
		}
	}
}
