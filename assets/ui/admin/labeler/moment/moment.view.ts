namespace $.$$ {
	export class $trip2g_admin_labeler_moment extends $.$trip2g_admin_labeler_moment {
		override formatted_value(): string {
			const v = this.value()
			if( !v ) {
				return '-'
			}

			const m = new $mol_time_moment( v )
			return m.toString( this.format() )
		}
	}
}
