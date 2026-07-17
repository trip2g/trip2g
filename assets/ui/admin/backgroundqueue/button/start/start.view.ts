namespace $.$$ {
	export class $trip2g_admin_backgroundqueue_button_start extends $.$trip2g_admin_backgroundqueue_button_start {
		override click() {
			const res = $trip2g_admin_backgroundqueue_button_start_mutate({ input: { id: this.queue_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
