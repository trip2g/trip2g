namespace $.$$ {
	export class $trip2g_admin_notfoundpattern_create extends $.$trip2g_admin_notfoundpattern_create {
		override body() {
			if( this.pattern_id_string() !== '' ) {
				return [ this.PatternView() ]
			}

			return super.body()
		}

		override pattern_bid(): string {
			const pattern = this.pattern()
			if( !pattern.trim() ) {
				return 'Pattern is required'
			}

			return ''
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateNotFoundIgnoredPatternMutation($input: CreateNotFoundIgnoredPatternInput!) {
						admin {
							data: createNotFoundIgnoredPattern(input: $input) {
								... on CreateNotFoundIgnoredPatternPayload {
									notFoundIgnoredPattern {
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
						pattern: this.pattern()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'CreateNotFoundIgnoredPatternPayload' ) {
				this.pattern_id_string( res.admin.data.notFoundIgnoredPattern.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}