namespace $.$$ {
	const data_request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminCronJobUpdate($id: Int64!) {
			admin {
				cronJob(id: $id) {
					id
					name
					enabled
					expression
					lastExecAt
				}
			}
		}
	`)

	const update_mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminUpdateCronJob($input: UpdateCronJobInput!) {
			admin {
				updateCronJob(input: $input) {
					__typename
					... on UpdateCronJobPayload {
						cronJob {
							id
							expression
							enabled
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)
	export class $trip2g_admin_cronjob_update extends $.$trip2g_admin_cronjob_update {
		@$mol_mem
		data(reset?: null) {
			const res = data_request({ id: this.cronjob_id() })

			if (!res.admin.cronJob) {
				throw new Error('Cron Job not found')
			}

			return res.admin.cronJob
		}

		cronjob_name(): string {
			return this.data().name
		}

		@$mol_mem
		expression(next?: string): string {
			if (next !== undefined) {
				return next
			}

			return this.data().expression || ''
		}

		@$mol_mem
		enabled(next?: boolean): boolean {
			if (next !== undefined) {
				return next
			}

			return this.data().enabled
		}

		expression_bid(): string {
			const expr = this.expression()
			if (!expr) return 'Expression is required'
			// Basic validation for cron expression format (6 parts for seconds-based cron)
			const parts = expr.trim().split(/\s+/)
			if (parts.length !== 6) {
				return 'Cron expression must have 6 parts (seconds, minutes, hours, day, month, day-of-week)'
			}
			return ''
		}

		enabled_bid(): string {
			return ''
		}

		submit_allowed(): boolean {
			return this.expression_bid() === '' && this.enabled_bid() === ''
		}

		submit() {
			const res = update_mutate({
				input: {
					id: this.cronjob_id(),
					expression: this.expression(),
					enabled: this.enabled()
				},
			})

			const result = res.admin.updateCronJob
			if (result.__typename === 'ErrorPayload') {
				this.result(result.message)
				return
			}

			if (result.__typename === 'UpdateCronJobPayload') {
				this.result('Cron Job updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}