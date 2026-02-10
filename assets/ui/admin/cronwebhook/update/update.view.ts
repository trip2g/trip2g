namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowCronWebhookForUpdate($id: Int64!) {
			admin {
				allCronWebhooks {
					nodes {
						id
						url
						cronSchedule
						enabled
						description
						instruction
						passApiKey
						maxDepth
						timeoutSeconds
						maxRetries
						readPatterns
						writePatterns
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminUpdateCronWebhookMutation($input: UpdateCronWebhookInput!) {
			admin {
				payload: updateCronWebhook(input: $input) {
					__typename
					... on UpdateCronWebhookPayload {
						cronWebhook {
							id
						}
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_cronwebhook_update extends $.$trip2g_admin_cronwebhook_update {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.cronwebhook_id() })
			const cw = res.admin.allCronWebhooks.nodes.find( (n: any) => n.id === this.cronwebhook_id() )
			if( !cw ) throw new Error( 'Cron Webhook not found' )
			return cw
		}

		@$mol_mem
		url(next?: string): string { return next ?? this.data().url ?? '' }

		@$mol_mem
		schedule(next?: string): string { return next ?? this.data().cronSchedule ?? '' }

		@$mol_mem
		description(next?: string): string { return next ?? this.data().description ?? '' }

		@$mol_mem
		instruction(next?: string): string { return next ?? this.data().instruction ?? '' }

		@$mol_mem
		enabled(next?: boolean): boolean { return next ?? this.data().enabled ?? true }

		@$mol_mem
		pass_api_key(next?: boolean): boolean { return next ?? this.data().passApiKey ?? false }

		@$mol_mem
		max_depth(next?: number): number { return next ?? this.data().maxDepth ?? 1 }

		@$mol_mem
		timeout_seconds(next?: number): number { return next ?? this.data().timeoutSeconds ?? 30 }

		@$mol_mem
		max_retries(next?: number): number { return next ?? this.data().maxRetries ?? 3 }

		override url_bid(): string {
			const url = this.url()
			if( !url.trim() ) return 'URL is required'
			try { new URL( url ) } catch { return 'Invalid URL' }
			return ''
		}

		override schedule_bid(): string {
			const schedule = this.schedule()
			if( !schedule.trim() ) return 'Cron schedule is required'
			return ''
		}

		submit() {
			const res = mutate({
				input: {
					id: this.cronwebhook_id(),
					url: this.url(),
					cronSchedule: this.schedule(),
					description: this.description() || undefined,
					instruction: this.instruction() || undefined,
					enabled: this.enabled(),
					passApiKey: this.pass_api_key(),
					maxDepth: this.max_depth(),
					timeoutSeconds: this.timeout_seconds(),
					maxRetries: this.max_retries(),
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'UpdateCronWebhookPayload' ) {
				this.result( 'Cron Webhook updated successfully' )
				this.data( null )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
