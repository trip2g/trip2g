namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminConfigValue($id: String!) {
			admin {
				configValue(id: $id) {
					__typename
					id
					description
					updatedAt
					... on AdminConfigStringValue {
						stringValue: value
						history {
							id
							value
							createdAt
							createdBy {
								email
							}
						}
					}
					... on AdminConfigBoolValue {
						boolValue: value
						boolHistory: history {
							id
							value
							createdAt
							createdBy {
								email
							}
						}
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

	export class $trip2g_admin_config_show extends $.$trip2g_admin_config_show {
		@$mol_mem
		data(reset?: null) {
			const res = data_request({ id: this.config_id() })
			if (!res.admin.configValue) {
				throw new Error('Config not found')
			}
			return res.admin.configValue
		}

		is_bool_config(): boolean {
			return this.data().__typename === 'AdminConfigBoolValue'
		}

		override config_title(): string {
			return this.config_id()
		}

		override config_description(): string {
			return this.data().description || ''
		}

		override config_current_value(): string {
			const data = this.data()
			if (data.__typename === 'AdminConfigBoolValue') {
				return data.boolValue ? 'true' : 'false'
			}
			return data.stringValue || ''
		}

		@$mol_mem
		override value_control() {
			if (this.is_bool_config()) {
				const control = new this.$.$mol_check_box()
				control.checked = (next?: boolean) => this.edit_value_bool(next)
				return control
			}
			const control = new this.$.$mol_string()
			control.value = (next?: string) => this.edit_value_string(next)
			return control
		}

		@$mol_mem
		edit_value_string(next?: string): string {
			if (next !== undefined) return next
			return this.data().stringValue || ''
		}

		@$mol_mem
		edit_value_bool(next?: boolean): boolean {
			if (next !== undefined) return next
			return this.data().boolValue || false
		}

		@$mol_mem
		result_message(next?: string): string {
			return next ?? ''
		}

		save() {
			const configId = this.config_id()

			if (this.is_bool_config()) {
				const value = this.edit_value_bool()
				const res = set_bool_value({ input: { id: configId, value } })
				const result = res.admin.setConfigBoolValue
				if (result.__typename === 'ErrorPayload') {
					this.result_message(result.message)
					return
				}
				this.result_message('Saved')
				this.data(null)
			} else {
				const value = this.edit_value_string()
				const res = set_string_value({ input: { id: configId, value } })
				const result = res.admin.setConfigStringValue
				if (result.__typename === 'ErrorPayload') {
					this.result_message(result.message)
					return
				}
				this.result_message('Saved')
				this.data(null)
			}
		}

		get_history() {
			const data = this.data()
			if (data.__typename === 'AdminConfigBoolValue') {
				return data.boolHistory || []
			}
			return data.history || []
		}

		@$mol_mem
		override history_rows() {
			return this.get_history().map((_, i) => this.HistoryRow(i))
		}

		history_entry(index: number) {
			return this.get_history()[index]
		}

		override history_value(index: any): string {
			const entry = this.history_entry(index)
			const value = entry.value
			if (typeof value === 'boolean') {
				return value ? 'true' : 'false'
			}
			return String(value || '')
		}

		override history_at(index: any): string {
			return this.history_entry(index).createdAt || ''
		}

		override history_by(index: any): string {
			return this.history_entry(index).createdBy?.email || '-'
		}
	}
}
