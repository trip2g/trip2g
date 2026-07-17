namespace $.$$ {
	export class $trip2g_admin_cronwebhook_show extends $.$trip2g_admin_cronwebhook_show {
		action() {
			return this.$.$mol_state_arg.value('action') || 'view'
		}

		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_admin_cronwebhook_show_data({ id: this.cronwebhook_id() })
			const cw = res.admin.cronWebhook
			if( !cw ) throw new Error( 'Cron Webhook not found' )
			return cw
		}

		@$mol_mem
		deliveries(reset?: null) {
			const res = $trip2g_admin_cronwebhook_show_deliveries({ id: this.cronwebhook_id() })
			return res.admin.cronWebhookDeliveries.nodes
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}
			return super.body()
		}

		cw_url(): string { return this.data().url }
		cw_enabled(): string { return this.data().enabled ? 'Yes' : 'No' }
		cw_description(): string { return this.data().description || '-' }
		cw_schedule(): string { return this.data().cronSchedule }

		cw_next_run_at(): string {
			const nra = this.data().nextRunAt
			if( !nra ) return '-'
			const m = new $mol_time_moment( nra )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		cw_instruction(): string { return this.data().instruction || '-' }
		cw_has_secret(): string { return this.data().hasSecret ? 'Yes' : 'No' }
		cw_pass_api_key(): string { return this.data().passApiKey ? 'Yes' : 'No' }
		cw_max_depth(): string { return this.data().maxDepth.toString() }
		cw_timeout_seconds(): string { return this.data().timeoutSeconds.toString() }
		cw_max_retries(): string { return this.data().maxRetries.toString() }
		cw_read_patterns(): string { return this.data().readPatterns.join( ', ' ) || '-' }
		cw_write_patterns(): string { return this.data().writePatterns.join( ', ' ) || '-' }

		cw_created_at(): string {
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

		secrets_prefix() {
			return this.data().secretPrefix
		}

		delete() {
			const res = $trip2g_admin_cronwebhook_show_delete({ input: { id: this.cronwebhook_id() } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.delete_result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'DeleteCronWebhookPayload' ) {
				this.delete_result( 'Cron Webhook deleted successfully' )
				this.$.$mol_state_arg.value( 'id', '' )
				return
			}

			this.delete_result( 'Unexpected response type' )
		}

		regenerate_secret() {
			const res = $trip2g_admin_cronwebhook_show_regenerate_secret({ input: { id: this.cronwebhook_id() } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.secret_result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'RegenerateCronWebhookSecretPayload' ) {
				this.secret_result( 'New secret: ' + res.admin.payload.secret )
				return
			}

			this.secret_result( 'Unexpected response type' )
		}

		trigger() {
			const res = $trip2g_admin_cronwebhook_show_trigger({ input: { cronWebhookId: this.cronwebhook_id() } })

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.trigger_result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'TriggerCronWebhookPayload' ) {
				this.trigger_result( `Webhook triggered successfully. Delivery ID: ${res.admin.payload.deliveryId}` )
				this.deliveries(null) // Refresh deliveries list
				return
			}

			this.trigger_result( 'Unexpected response type' )
		}
	}
}
