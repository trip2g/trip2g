namespace $.$$ {
	export class $trip2g_admin_notfoundpattern_update extends $.$trip2g_admin_notfoundpattern_update {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminShowNotFoundIgnoredPattern {
						admin {
							allNotFoundIgnoredPatterns {
								nodes {
									id
									pattern
									createdAt
									createdBy {
										id
										email
									}
								}
							}
						}
					}
				`
			)

			const pattern = res.admin.allNotFoundIgnoredPatterns.nodes.find((n: any) => n.id === this.pattern_id())
			if (!pattern) {
				throw new Error('Ignored Pattern not found')
			}

			return pattern
		}

		override pattern(): string {
			return this.data().pattern
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
					mutation AdminUpdateNotFoundIgnoredPatternMutation($input: UpdateNotFoundIgnoredPatternInput!) {
						admin {
							data: updateNotFoundIgnoredPattern(input: $input) {
								... on UpdateNotFoundIgnoredPatternPayload {
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
						id: this.pattern_id(),
						pattern: this.pattern()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'UpdateNotFoundIgnoredPatternPayload' ) {
				this.result( 'Pattern updated successfully' )
				// Navigate back to show page
				this.$.$mol_state_arg.value('action', '')
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}