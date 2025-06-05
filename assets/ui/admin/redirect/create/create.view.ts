namespace $.$$ {
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
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateRedirectMutation($input: CreateRedirectInput!) {
						admin {
							data: createRedirect(input: $input) {
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
				`,
				{
					input: {
						pattern: this.pattern(),
						target: this.target(),
						isRegex: this.is_regex(),
						ignoreCase: this.ignore_case()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'CreateRedirectPayload' ) {
				this.redirect_id_string( res.admin.data.redirect.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}