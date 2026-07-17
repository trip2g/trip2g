namespace $.$$ {
	export class $trip2g_admin_htmlinjection_button_delete extends $.$trip2g_admin_htmlinjection_button_delete {
		click(e: PointerEvent) {
			e.stopPropagation()
			e.preventDefault()

			const res = $trip2g_admin_htmlinjection_button_delete_delete({
				input: {
					id: this.htmlinjection_id(),
				},
			});

			if (res.admin.data.__typename === 'ErrorPayload') {
				throw new Error(res.admin.data.message);
			}

			if (res.admin.data.__typename === 'DeleteHtmlInjectionPayload') {
				this.after_success()
			}
		}
	}
}
