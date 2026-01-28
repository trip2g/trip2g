namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminConfigValues {
			admin {
				configValues {
					__typename
					id
					description
					updatedAt
					updatedBy {
						email
					}
					... on AdminConfigStringValue {
						stringValue: value
					}
					... on AdminConfigBoolValue {
						boolValue: value
					}
				}
			}
		}
	`)

	export class $trip2g_admin_config_catalog extends $.$trip2g_admin_config_catalog {
		@$mol_mem
		data(reset?: null) {
			const res = data_request()
			return $trip2g_graphql_make_map(res.admin.configValues)
		}

		@$mol_mem
		spreads(): any {
			return this.data().mapKeys(key => this.Content(key))
		}

		row(id: any) {
			return this.data().get(id)
		}

		override row_id(id: any): string {
			return this.row(id).id
		}

		override row_value_preview(id: any): string {
			const row = this.row(id)
			let value: string
			if (row.__typename === 'AdminConfigBoolValue') {
				value = row.boolValue ? 'true' : 'false'
			} else {
				value = row.stringValue || ''
			}
			// Truncate long values.
			if (value.length > 50) {
				return value.substring(0, 47) + '...'
			}
			return value
		}

		override row_updated_at(id: any): string {
			return this.row(id).updatedAt || ''
		}

		override row_updated_by(id: any): string {
			return this.row(id).updatedBy?.email || '-'
		}
	}
}
