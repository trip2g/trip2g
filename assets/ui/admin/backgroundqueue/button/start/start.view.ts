namespace $.$$ {
	const mutate = $trip2g_graphql_request( /* GraphQL */ `
		mutation AdminStartBackgroundQueue($input: StartBackgroundQueueInput!) {
			admin {
				payload: startBackgroundQueue(input: $input) {
					... on StartBackgroundQueuePayload {
						queues {
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

	export class $trip2g_admin_backgroundqueue_button_start extends $.$trip2g_admin_backgroundqueue_button_start {
		override click() {
			const res = mutate({ input: { id: this.queue_id() } })

			if (res.admin.payload.__typename === 'ErrorPayload') {
				throw new Error(res.admin.payload.message)
				return
			}
		}
	}
}
