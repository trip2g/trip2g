namespace $.$$ {
	export class $trip2g_admin_backgroundqueue_button_stop extends $.$trip2g_admin_backgroundqueue_button_stop {
		override click() {
			const res = $trip2g_admin_backgroundqueue_button_stop_mutate({ input: { id: this.queue_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
