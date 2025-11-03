namespace $.$$ {
	export class $trip2g_admin_labeler_moment extends $.$trip2g_admin_labeler_moment {
		override formatted_value(): string {
			const m = new $mol_time_moment( this.value() )
			return m.toString( this.format() )
		}
	}
}
