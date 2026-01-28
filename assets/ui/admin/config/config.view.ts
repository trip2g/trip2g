namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminConfigValues {
			admin {
				configValues {
					__typename
					id
					description
					updatedAt
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

	const set_string_value = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminSetConfigStringValue($input: SetConfigStringValueInput!) {
			admin {
				setConfigStringValue(input: $input) {
					__typename
					... on SetConfigStringValueSuccess {
						configValue {
							id
							value
							updatedAt
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	const set_bool_value = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminSetConfigBoolValue($input: SetConfigBoolValueInput!) {
			admin {
				setConfigBoolValue(input: $input) {
					__typename
					... on SetConfigBoolValueSuccess {
						configValue {
							id
							value
							updatedAt
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_config extends $.$trip2g_admin_config {
		@$mol_mem
		data(reset?: null) {
			const res = data_request()
			return $trip2g_graphql_make_map(res.admin.configValues)
		}

		@$mol_mem
		override rows() {
			return this.data().map(id => this.Row(id))
		}

		row(id: string) {
			const row = this.data().get(id)
			if (!row) throw new Error('Config not found')
			return row
		}

		override row_id(id: any): string {
			return this.row(id).id
		}

		override row_description(id: any): string {
			return this.row(id).description || ''
		}

		override row_updated_at(id: any): string {
			const updatedAt = this.row(id).updatedAt
			if (!updatedAt) return '-'
			const m = new $mol_time_moment(updatedAt)
			return m.toString('YYYY-MM-DD hh:mm')
		}

		is_bool_config(id: any): boolean {
			return this.row(id).__typename === 'AdminConfigBoolValue'
		}

		@$mol_mem_key
		override value_control(id: any) {
			if (this.is_bool_config(id)) {
				return this.value_bool_control(id)
			}
			return this.value_string_control(id)
		}

		value_string_control(id: any) {
			const control = new this.$.$mol_string()
			control.value = (next?: string) => this.row_value_string(id, next)
			return control
		}

		value_bool_control(id: any) {
			const control = new this.$.$mol_check_box()
			control.checked = (next?: boolean) => this.row_value_bool(id, next)
			return control
		}

		@$mol_mem_key
		row_value_string(id: any, next?: string): string {
			if (next !== undefined) return next
			const row = this.row(id)
			return row.stringValue || ''
		}

		@$mol_mem_key
		row_value_bool(id: any, next?: boolean): boolean {
			if (next !== undefined) return next
			const row = this.row(id)
			return row.boolValue || false
		}

		@$mol_mem_key
		row_status_message(id: any, next?: string): string {
			return next ?? ''
		}

		save_row(id: any) {
			const configId = this.row_id(id)

			if (this.is_bool_config(id)) {
				const value = this.row_value_bool(id)
				const res = set_bool_value({
					input: { id: configId, value }
				})
				const result = res.admin.setConfigBoolValue
				if (result.__typename === 'ErrorPayload') {
					this.row_status_message(id, result.message)
					return
				}
				this.row_status_message(id, 'Saved')
				this.data(null)
			} else {
				const value = this.row_value_string(id)
				const res = set_string_value({
					input: { id: configId, value }
				})
				const result = res.admin.setConfigStringValue
				if (result.__typename === 'ErrorPayload') {
					this.row_status_message(id, result.message)
					return
				}
				this.row_status_message(id, 'Saved')
				this.data(null)
			}
		}
	}
}
