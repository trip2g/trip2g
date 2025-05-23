namespace $.$$ {
	export class $trip2g_table_cell_time extends $.$trip2g_table_cell_time {
		override formatted_value(): string {
			const v = this.value()
			if (!v) {
				return '-'
			}

			const m = new this.$.$mol_time_moment(v)
			return m.toString('DD.MM.YYYY')
		}
	}
}