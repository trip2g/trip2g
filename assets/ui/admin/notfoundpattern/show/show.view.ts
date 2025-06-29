namespace $.$$ {
	export class $trip2g_admin_notfoundpattern_show extends $.$trip2g_admin_notfoundpattern_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data( reset?: null ) {
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

			const pattern = res.admin.allNotFoundIgnoredPatterns.nodes.find( ( n: any ) => n.id === this.pattern_id() )
			if( !pattern ) {
				throw new Error( 'Ignored Pattern not found' )
			}

			return pattern
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return [ this.PatternDetails(), this.DeleteResult() ]
		}

		pattern_pattern(): string {
			return this.data().pattern
		}

		pattern_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		pattern_created_by(): string {
			return this.data().createdBy.email || '-'
		}

		delete() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminDeleteNotFoundIgnoredPatternMutation($input: DeleteNotFoundIgnoredPatternInput!) {
						admin {
							data: deleteNotFoundIgnoredPattern(input: $input) {
								... on DeleteNotFoundIgnoredPatternPayload {
									deletedId
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
						id: this.pattern_id()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.delete_result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'DeleteNotFoundIgnoredPatternPayload' ) {
				this.delete_result( 'Ignored Pattern deleted successfully' )
				// Navigate back to catalog
				this.$.$mol_state_arg.value( 'id', '' )
				return
			}

			this.delete_result( 'Unexpected response type' )
		}
	}
}