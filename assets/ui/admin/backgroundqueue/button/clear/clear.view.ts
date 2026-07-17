namespace $.$$ {
	export class $trip2g_admin_backgroundqueue_button_clear extends $.$trip2g_admin_backgroundqueue_button_clear {
		override click() {
			const res = $trip2g_admin_backgroundqueue_button_clear_mutate({ input: { id: this.queue_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}

			if (res.admin.payload.__typename === 'ClearBackgroundQueuePayload') {
				// Show success message with deleted count
				const { deletedCount } = res.admin.payload
				this.$.$mol_log3_rise({
					message: `Successfully cleared queue. Deleted ${deletedCount} job(s).`,
					place: this,
				})
			}
		}
	}
}
