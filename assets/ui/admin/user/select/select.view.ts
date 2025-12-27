namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminSelectUser {
			admin {
				allUsers {
					nodes {
						id
						email
					}
				}
			}
		}
	`)

	export class $trip2g_admin_user_select extends $.$trip2g_admin_user_select {
		@$mol_mem
		data(reset?: null) {
			const res = request()

			return res.admin.allUsers.nodes
		}

		dictionary(): Record<string, string> {
			const map: { [id: number]: string } = {}

			this.data().forEach((row) => {
				map[row.id] = row.email || `User #${row.id}`
			})

			return map
		}

		@$mol_mem
		value(next?: string): string {
			if (next === undefined) {
				const n = this.number_value()
				return n ? n.toString() : ''
			}

			if (next) {
				this.number_value(parseInt(next, 10))
			} else {
				this.number_value(null)
			}

			return next || ''
		}
	}
}
