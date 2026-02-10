namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */`
		query AdminShowChangeWebhook($id: Int64!) {
			admin {
				changeWebhookDeliveries(filter: { webhookId: $id, limit: 20 }) {
					nodes {
						id
						status
						responseStatus
						attempt
						durationMs
						createdAt
						completedAt
					}
				}
			}
		}
	`)

	const webhookQuery = $trip2g_graphql_request(/* GraphQL */`
		query AdminGetChangeWebhook($id: Int64!) {
			admin {
				allChangeWebhooks {
					nodes {
						id
						url
						enabled
						description
						instruction
						hasSecret
						passApiKey
						includeContent
						maxDepth
						timeoutSeconds
						maxRetries
						onCreate
						onUpdate
						onRemove
						includePatterns
						excludePatterns
						readPatterns
						writePatterns
						createdAt
					}
				}
			}
		}
	`)

	const deleteMutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminDeleteWebhookMutation($input: DeleteWebhookInput!) {
			admin {
				payload: deleteWebhook(input: $input) {
					__typename
					... on DeleteWebhookPayload {
						deletedId
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	const regenerateMutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminRegenerateWebhookSecretMutation($input: RegenerateWebhookSecretInput!) {
			admin {
				payload: regenerateWebhookSecret(input: $input) {
					__typename
					... on RegenerateWebhookSecretPayload {
						secret
					}
					... on ErrorPayload {
						message
					}
				}
			}
		}
	`)

	export class $trip2g_admin_webhook_show extends $.$trip2g_admin_webhook_show {
		action() {
			return this.$.$mol_state_arg.value('action') || 'view'
		}

		@$mol_mem
		data(reset?: null) {
			const res = webhookQuery({ id: this.webhook_id() })
			const wh = res.admin.allChangeWebhooks.nodes.find( (n: any) => n.id === this.webhook_id() )
			if( !wh ) throw new Error( 'Webhook not found' )
			return wh
		}

		@$mol_mem
		deliveries(reset?: null) {
			const res = request({ id: this.webhook_id() })
			return res.admin.changeWebhookDeliveries.nodes
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}
			return [ this.WebhookDetails(), this.SecretResult(), this.DeleteResult(), this.Deliveries_labeler() ]
		}

		webhook_url(): string { return this.data().url }
		webhook_enabled(): string { return this.data().enabled ? 'Yes' : 'No' }
		webhook_description(): string { return this.data().description || '-' }

		webhook_events(): string {
			const d = this.data()
			const events: string[] = []
			if( d.onCreate ) events.push( 'create' )
			if( d.onUpdate ) events.push( 'update' )
			if( d.onRemove ) events.push( 'remove' )
			return events.join( ', ' ) || 'none'
		}

		webhook_include_patterns(): string { return this.data().includePatterns.join( ', ' ) || '-' }
		webhook_exclude_patterns(): string { return this.data().excludePatterns.join( ', ' ) || '-' }
		webhook_instruction(): string { return this.data().instruction || '-' }
		webhook_has_secret(): string { return this.data().hasSecret ? 'Yes' : 'No' }
		webhook_pass_api_key(): string { return this.data().passApiKey ? 'Yes' : 'No' }
		webhook_include_content(): string { return this.data().includeContent ? 'Yes' : 'No' }
		webhook_max_depth(): string { return this.data().maxDepth.toString() }
		webhook_timeout_seconds(): string { return this.data().timeoutSeconds.toString() }
		webhook_max_retries(): string { return this.data().maxRetries.toString() }
		webhook_read_patterns(): string { return this.data().readPatterns.join( ', ' ) || '-' }
		webhook_write_patterns(): string { return this.data().writePatterns.join( ', ' ) || '-' }

		webhook_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		@$mol_mem
		delivery_rows() {
			return this.deliveries().map( (_: any, i: number) => this.DeliveryRow( i ) )
		}

		delivery_id( index: number ): string { return this.deliveries()[ index ].id.toString() }
		delivery_status( index: number ): string { return this.deliveries()[ index ].status }

		delivery_response( index: number ): string {
			const s = this.deliveries()[ index ].responseStatus
			return s ? s.toString() : '-'
		}

		delivery_attempt( index: number ): string { return this.deliveries()[ index ].attempt.toString() }

		delivery_duration( index: number ): string {
			const ms = this.deliveries()[ index ].durationMs
			return ms ? `${ms}ms` : '-'
		}

		delivery_created_at( index: number ): string {
			const m = new $mol_time_moment( this.deliveries()[ index ].createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm:ss' )
		}

		delete() {
			const res = deleteMutate({ input: { id: this.webhook_id() } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.delete_result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'DeleteWebhookPayload' ) {
				this.delete_result( 'Webhook deleted successfully' )
				this.$.$mol_state_arg.value( 'id', '' )
				return
			}

			this.delete_result( 'Unexpected response type' )
		}

		regenerate_secret() {
			const res = regenerateMutate({ input: { id: this.webhook_id() } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.secret_result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'RegenerateWebhookSecretPayload' ) {
				this.secret_result( 'New secret: ' + res.admin.payload.secret )
				return
			}

			this.secret_result( 'Unexpected response type' )
		}
	}
}
