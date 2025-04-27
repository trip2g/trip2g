namespace $.$$ {
	export class $trip2g_admin_list_userbans extends $.$trip2g_admin_list_userbans {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(`
				query AdminListUserBans {
					admin {
						allUserUserBans {
							nodes {
								id: userId
								user {
									email
								}
								bannedBy {
									user {
										email
									}
								}
								createdAt
								reason
							}
						}
					}
				}
			`)

			return $trip2g_graphql_make_map(res.admin.allUserUserBans.nodes)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key));
		}

		body() {
			return this.data().map(key => this.Row(key));
		}

		row(id: any) {
			return this.data().get(id);
		}

		row_user_email(id: any): string {
			return this.row(id).user.email;
		}

		row_banned_by_email(id: any): string {
			return this.row(id).bannedBy?.user.email || '-';
		}

		row_created_at(id: any): string {
			const m = new $mol_time_moment(this.row(id).createdAt)
			return m.toString('YYYY-MM-DD')
		}

		row_reason(id: any): string {
			return this.row(id).reason;
		}
	}
}
