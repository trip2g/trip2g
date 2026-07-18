namespace $.$$ {
	export class $trip2g_admin_notfoundpath_button_reset extends $.$trip2g_admin_notfoundpath_button_reset {
		click() {
			const res = $trip2g_admin_notfoundpath_button_reset_reset({
				input: {
					id: this.notfoundpath_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}
		}
	}
}