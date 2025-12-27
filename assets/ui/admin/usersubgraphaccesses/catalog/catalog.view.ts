namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminListUserSubgraphAccesses {
			admin {
				data: allUserSubgraphAccesses {
					nodes {
						__typename
						id
						createdAt
						expiresAt
						subgraph {
							name
						}
						user {
							id
							email
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_usersubgraphaccesses_catalog extends $.$trip2g_admin_usersubgraphaccesses_catalog {
		@$mol_mem
		data(reset?: null) {
			const res = request()

			return $trip2g_graphql_make_map(res.admin.data.nodes)
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.CreateForm(),
				...this.data().mapKeys(key => this.ShowPage(key))
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter(id => id !== 'add')
		}

		row(id: any) {
			return this.data().get(id)
		}

		row_id(id: any): number {
			return this.row(id).id
		}

		row_id_string(id: any): string {
			return this.row(id).id.toString()
		}

		row_subgraph_name(id: any): string {
			return this.row(id).subgraph.name
		}

		row_created_at(id: any): string {
			const m = new $mol_time_moment(this.row(id).createdAt)
			return m.toString('YYYY-MM-DD')
		}

		row_expires_at(id: any): string {
			const raw = this.row(id).expiresAt
			if (raw) {
				return new $mol_time_moment(raw).toString('YYYY-MM-DD')
			}

			return '-'
		}

		row_user_email(id: any): string {
			return this.row(id).user.email || '-'
		}

		row_user_uri(id: any): string {
			return this.$.$mol_state_arg.link({
				nav: 'users',
				id: this.row(id).user.id,
			})
		}
	}
}
