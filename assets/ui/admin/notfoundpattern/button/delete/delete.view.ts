namespace $.$$ {
	export class $trip2g_admin_notfoundpattern_button_delete extends $.$trip2g_admin_notfoundpattern_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_admin_notfoundpattern_button_delete_delete({
				input: {
					id: this.pattern_id(),
				},
			});

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message);
			}
		}
	}
}