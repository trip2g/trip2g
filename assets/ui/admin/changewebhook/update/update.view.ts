namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowChangeWebhookForUpdate($id: Int64!) {
			admin {
				allChangeWebhooks {
					nodes {
						id
						url
						enabled
						description
						instruction
						includePatterns
						excludePatterns
						passApiKey
						includeContent
						maxDepth
						timeoutSeconds
						maxRetries
						onCreate
						onUpdate
						onRemove
						readPatterns
						writePatterns
					}
				}
			}
		}
	`)

	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminUpdateWebhookMutation($input: UpdateWebhookInput!) {
			admin {
				payload: updateWebhook(input: $input) {
					__typename
					... on UpdateWebhookPayload {
						webhook {
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

	export class $trip2g_admin_changewebhook_update extends $.$trip2g_admin_changewebhook_update {
		@$mol_mem
		data(reset?: null) {
			const res = request({ id: this.changewebhook_id() })
			const wh = res.admin.allChangeWebhooks.nodes.find( (n: any) => n.id === this.changewebhook_id() )
			if( !wh ) throw new Error( 'Webhook not found' )
			return wh
		}

		parse_patterns( raw: string ): string[] {
			return raw.split( ',' ).map( s => s.trim() ).filter( s => s.length > 0 )
		}

		@$mol_mem
		url(next?: string): string { return next ?? this.data().url ?? '' }

		@$mol_mem
		include_patterns_raw(next?: string): string { return next ?? this.data().includePatterns.join( ', ' ) ?? '' }

		@$mol_mem
		exclude_patterns_raw(next?: string): string { return next ?? this.data().excludePatterns.join( ', ' ) ?? '' }

		@$mol_mem
		description(next?: string): string { return next ?? this.data().description ?? '' }

		@$mol_mem
		instruction(next?: string): string { return next ?? this.data().instruction ?? '' }

		@$mol_mem
		enabled(next?: boolean): boolean { return next ?? this.data().enabled ?? true }

		@$mol_mem
		on_create(next?: boolean): boolean { return next ?? this.data().onCreate ?? true }

		@$mol_mem
		on_update(next?: boolean): boolean { return next ?? this.data().onUpdate ?? true }

		@$mol_mem
		on_remove(next?: boolean): boolean { return next ?? this.data().onRemove ?? false }

		@$mol_mem
		pass_api_key(next?: boolean): boolean { return next ?? this.data().passApiKey ?? false }

		@$mol_mem
		include_content(next?: boolean): boolean { return next ?? this.data().includeContent ?? true }

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

		override include_patterns_bid(): string {
			const raw = this.include_patterns_raw()
			if( !raw.trim() ) return 'At least one include pattern is required'
			return ''
		}

		submit() {
			const res = mutate({
				input: {
					id: this.changewebhook_id(),
					url: this.url(),
					includePatterns: this.parse_patterns( this.include_patterns_raw() ),
					excludePatterns: this.parse_patterns( this.exclude_patterns_raw() ),
					description: this.description() || undefined,
					instruction: this.instruction() || undefined,
					enabled: this.enabled(),
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

			if( res.admin.payload.__typename === 'UpdateWebhookPayload' ) {
				this.result( 'Webhook updated successfully' )
				this.data( null )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
