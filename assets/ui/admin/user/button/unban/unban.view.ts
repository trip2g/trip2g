namespace $.$$ {
	export class $trip2g_admin_user_button_unban extends $.$trip2g_admin_user_button_unban {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_admin_user_button_unban_save({
				input: {
					userId: this.user_id(),
				},
			});

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message);
			}
		}
	}
}
