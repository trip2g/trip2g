namespace $.$$ {
	const TIMEZONE_SUGGESTS = [
		'UTC',
		'America/New_York',
		'America/Los_Angeles',
		'America/Chicago',
		'Europe/London',
		'Europe/Paris',
		'Europe/Moscow',
		'Asia/Dubai',
		'Asia/Kolkata',
		'Asia/Shanghai',
		'Asia/Tokyo',
		'Asia/Singapore',
		'Australia/Sydney',
		'Pacific/Auckland',
	]

	export class $trip2g_admin_config_show extends $.$trip2g_admin_config_show {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_config_show_data({ id: this.config_id() })
			if (!res.admin.configValue) {
				throw new Error('Config not found')
			}
			return res.admin.configValue
		}

		is_bool_config(): boolean {
			return this.data().__typename === 'AdminConfigBoolValue'
		}

		is_int_config(): boolean {
			return this.data().__typename === 'AdminConfigIntValue'
		}

		is_timezone_config(): boolean {
			return this.config_id() === 'timezone'
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
			if (data.__typename === 'AdminConfigIntValue') {
				return String(data.intValue)
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
			if (this.is_int_config()) {
				const control = new this.$.$mol_number()
				control.value = (next?: number) => this.edit_value_int(next)
				return control
			}
			if (this.is_timezone_config()) {
				const control = new this.$.$mol_search()
				control.query = (next?: string) => this.edit_value_string(next)
				control.suggests = () => TIMEZONE_SUGGESTS
				control.hint = () => 'Enter timezone or select from list'
				return control
			}
			const control = new this.$.$mol_string()
			control.value = (next?: string) => this.edit_value_string(next)
			return control
		}

		@$mol_mem
		override value_field_content() {
			const controls: any[] = [this.value_control()]
			if (this.is_timezone_config()) {
				controls.push(this.MyTimezoneButton())
			}
			return controls
		}

		override my_timezone(): string {
			return Intl.DateTimeFormat().resolvedOptions().timeZone
		}

		set_my_timezone() {
			this.edit_value_string(this.my_timezone())
		}

		@$mol_mem
		edit_value_string(next?: string): string {
			if (next !== undefined) return next

			const data = this.data()

			if (data.__typename !== 'AdminConfigStringValue') {
				throw new Error('Not a string config')
			}

			return data.stringValue || ''
		}

		@$mol_mem
		edit_value_bool(next?: boolean): boolean {
			if (next !== undefined) return next

			const data = this.data()

			if (data.__typename !== 'AdminConfigBoolValue') {
				throw new Error('Not a bool config')
			}

			return data.boolValue || false
		}

		@$mol_mem
		edit_value_int(next?: number): number {
			if (next !== undefined) return next

			const data = this.data()

			if (data.__typename !== 'AdminConfigIntValue') {
				throw new Error('Not an int config')
			}

			return data.intValue || 0
		}

		@$mol_mem
		result_message(next?: string): string {
			return next ?? ''
		}

		save() {
			const configId = this.config_id()

			if (this.is_bool_config()) {
				const value = this.edit_value_bool()
				const res = $trip2g_admin_config_show_save_bool({ input: { id: configId, value } })
				const result = res.admin.setConfigBoolValue
				if (result.__typename === 'ErrorPayload') {
					this.result_message(result.message)
					return
				}
				this.result_message('Saved')
				this.data(null)
			} else if (this.is_int_config()) {
				const value = this.edit_value_int()
				const res = $trip2g_admin_config_show_save_int({ input: { id: configId, value } })
				const result = res.admin.setConfigIntValue
				if (result.__typename === 'ErrorPayload') {
					this.result_message(result.message)
					return
				}
				this.result_message('Saved')
				this.data(null)
			} else {
				const value = this.edit_value_string()
				const res = $trip2g_admin_config_show_save_string({ input: { id: configId, value } })
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
			if (data.__typename === 'AdminConfigIntValue') {
				return data.intHistory || []
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
