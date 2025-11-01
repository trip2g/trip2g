namespace $.$$ {
	const mutate = $trip2g_graphql_request( /* GraphQL */ `
		mutation AdminStopBackgroundQueue($input: StopBackgroundQueueInput!) {
			admin {
				payload: stopBackgroundQueue(input: $input) {
					... on StopBackgroundQueuePayload {
						queue {
							id
							stopped
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_backgroundqueue_button_stop extends $.$trip2g_admin_backgroundqueue_button_stop {
		override click() {
			const res = mutate({ input: { id: this.queue_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
