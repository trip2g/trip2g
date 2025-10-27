namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminCreateNotFoundIgnoredPatternMutation($input: CreateNotFoundIgnoredPatternInput!) {
			admin {
				payload: createNotFoundIgnoredPattern(input: $input) {
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
	`)

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
			const res = mutate({
				input: {
					pattern: this.pattern()
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'CreateNotFoundIgnoredPatternPayload' ) {
				this.pattern_id_string( res.admin.payload.notFoundIgnoredPattern.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}