namespace $.$$ {
	const notePathsQuery = $trip2g_graphql_request(/* GraphQL */`
		query AllNotePaths {
			notePaths {
				id
				value
			}
		}
	`)

	const triggerMutation = $trip2g_graphql_request(/* GraphQL */`
		mutation TriggerChangeWebhook($input: TriggerChangeWebhookInput!) {
			admin {
				payload: triggerChangeWebhook(input: $input) {
					__typename
					... on TriggerChangeWebhookPayload {
						matchedCount
						ignoredCount
						deliveryId
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_changewebhook_trigger extends $.$trip2g_admin_changewebhook_trigger {
		@$mol_mem
		note_paths(reset?: null) {
			const res = notePathsQuery()
			return res.notePaths
		}

		@$mol_mem
		path_dictionary(): Record<string, string> {
			const dict: Record<string, string> = {}
			this.note_paths().forEach((path: any) => {
				dict[path.id] = path.value
			})
			return dict
		}

		@$mol_mem
		selected_path_ids(next?: string[]): string[] {
			if (next !== undefined) {
				return next
			}
			return []
		}

		trigger() {
			const pathIds = this.selected_path_ids().map(id => id.toString())

			if (pathIds.length === 0) {
				this.trigger_result('Please select at least one path')
				return
			}

			try {
				const res = triggerMutation({
					input: {
						webhookId: this.changewebhook_id().toString(),
						pathIds: pathIds
					}
				})

				const { payload } = res.admin

				if (payload.__typename === 'ErrorPayload') {
					this.trigger_result(payload.message)
					return
				}

				if (payload.__typename === 'TriggerChangeWebhookPayload') {
					const msg = `Success! Matched: ${payload.matchedCount}, Ignored: ${payload.ignoredCount}, Delivery ID: ${payload.deliveryId}`
					this.trigger_result(msg)

					// Navigate back to show page after 2 seconds
					setTimeout(() => {
						this.$.$mol_state_arg.value('action', 'view')
					}, 2000)
					return
				}

				this.trigger_result('Unexpected response type')
			} catch (error) {
				this.trigger_result(`Error: ${(error as Error).message}`)
			}
		}
	}
}
