namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateRedirectMutation($input: CreateRedirectInput!) {
			admin {
				payload: createRedirect(input: $input) {
					... on CreateRedirectPayload {
						redirect {
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

	export class $trip2g_admin_redirect_create extends $.$trip2g_admin_redirect_create {
		override body() {
			if( this.redirect_id_string() !== '' ) {
				return [ this.RedirectView() ]
			}

			return super.body()
		}

		override pattern_bid(): string {
			const pattern = this.pattern()
			if( !pattern.trim() ) {
				return 'Pattern is required'
			}

			if( this.is_regex() ) {
				try {
					new RegExp( pattern )
				} catch( e ) {
					return 'Invalid regex pattern'
				}
			}

			return ''
		}

		override target_bid(): string {
			const target = this.target()
			if( !target.trim() ) {
				return 'Target is required'
			}

			return ''
		}

		submit() {
			const res = mutate({
				input: {
					pattern: this.pattern(),
					target: this.target(),
					isRegex: this.is_regex(),
					ignoreCase: this.ignore_case()
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateRedirectPayload' ) {
				this.redirect_id_string( res.admin.payload.redirect.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}