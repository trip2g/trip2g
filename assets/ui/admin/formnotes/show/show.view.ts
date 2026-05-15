namespace $.$$ {
	export class $trip2g_admin_formnotes_show extends $.$trip2g_admin_formnotes_show {
		override note_path( val = 0 ): string {
			return `Form Note #${ this.note_path_id() }`
		}
	}
}
