namespace $.$$ {
	export class $trip2g_admin_telegrampublishnote_button_send extends $.$trip2g_admin_telegrampublishnote_button_send {
		override click() {
			const res = $trip2g_admin_telegrampublishnote_button_send_send({ input: { id: this.note_path_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
