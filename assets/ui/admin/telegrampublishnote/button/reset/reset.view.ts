namespace $.$$ {
	export class $trip2g_admin_telegrampublishnote_button_reset extends $.$trip2g_admin_telegrampublishnote_button_reset {
		override click() {
			const res = $trip2g_admin_telegrampublishnote_button_reset_reset({ input: { id: this.note_path_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
