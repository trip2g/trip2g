namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateWebhookMutation($input: CreateWebhookInput!) {
			admin {
				payload: createWebhook(input: $input) {
					__typename
					... on CreateWebhookPayload {
						webhook {
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

	export class $trip2g_admin_changewebhook_create extends $.$trip2g_admin_changewebhook_create {
		override body() {
			if( this.created_id_string() !== '' ) {
				return [ this.CreatedView() ]
			}
			return super.body()
		}

		override url_bid(): string {
			const url = this.url()
			if( !url.trim() ) return 'URL is required'
			try {
				new URL( url )
			} catch {
				return 'Invalid URL'
			}
			return ''
		}

		override include_patterns_bid(): string {
			const raw = this.include_patterns_raw()
			if( !raw.trim() ) return 'At least one include pattern is required'
			return ''
		}

		parse_patterns( raw: string ): string[] {
			return raw.split( ',' ).map( s => s.trim() ).filter( s => s.length > 0 )
		}

		submit() {
			const res = mutate({
				input: {
					url: this.url(),
					includePatterns: this.parse_patterns( this.include_patterns_raw() ),
					excludePatterns: this.parse_patterns( this.exclude_patterns_raw() ),
					description: this.description() || undefined,
					instruction: this.instruction() || undefined,
					onCreate: this.on_create(),
					onUpdate: this.on_update(),
					onRemove: this.on_remove(),
					passApiKey: this.pass_api_key(),
					includeContent: this.include_content(),
					maxDepth: this.max_depth(),
					timeoutSeconds: this.timeout_seconds(),
					maxRetries: this.max_retries(),
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateWebhookPayload' ) {
				this.created_id_string( res.admin.payload.webhook.id.toString() )
				this.created_secret( res.admin.payload.secret )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
