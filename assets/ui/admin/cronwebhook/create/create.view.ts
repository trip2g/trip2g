namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateCronWebhookMutation($input: CreateCronWebhookInput!) {
			admin {
				payload: createCronWebhook(input: $input) {
					__typename
					... on CreateCronWebhookPayload {
						cronWebhook {
							id
						}
						secret
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_cronwebhook_create extends $.$trip2g_admin_cronwebhook_create {
		override body() {
			if( this.created_id_string() !== '' ) {
				return [ this.CreatedView() ]
			}
			return super.body()
		}

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
					url: this.url(),
					cronSchedule: this.schedule(),
					description: this.description() || undefined,
					instruction: this.instruction() || undefined,
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

			if( res.admin.payload.__typename === 'CreateCronWebhookPayload' ) {
				this.created_id_string( res.admin.payload.cronWebhook.id.toString() )
				this.created_secret( res.admin.payload.secret )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
