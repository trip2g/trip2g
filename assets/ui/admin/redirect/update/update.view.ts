namespace $.$$ {
	export class $trip2g_admin_redirect_update extends $.$trip2g_admin_redirect_update {
		@$mol_mem
		data(reset?: null) {
			const res = $trip2g_graphql_request(
				`
					query AdminShowRedirect($id: Int64!) {
						admin {
							redirect(id: $id) {
								id
								createdAt
								pattern
								ignoreCase
								isRegex
								target
								createdBy {
									id
									email
								}
							}
						}
					}
				`,
				{ id: this.redirect_id() }
			)

			if (!res.admin.redirect) {
				throw new Error('Redirect not found')
			}

			return res.admin.redirect
		}

		redirect_id_string(): string {
			return this.data().id.toString()
		}

		@$mol_mem
		pattern(next?: string): string {
			return next ?? this.data().pattern ?? ''
		}

		@$mol_mem
		target(next?: string): string {
			return next ?? this.data().target ?? ''
		}

		@$mol_mem
		is_regex(next?: boolean): boolean {
			return next ?? this.data().isRegex ?? false
		}

		@$mol_mem
		ignore_case(next?: boolean): boolean {
			return next ?? this.data().ignoreCase ?? true
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
					mutation AdminUpdateRedirectMutation($input: UpdateRedirectInput!) {
						admin {
							data: updateRedirect(input: $input) {
								... on UpdateRedirectPayload {
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
						id: this.redirect_id(),
						pattern: this.pattern(),
						target: this.target(),
						isRegex: this.is_regex(),
						ignoreCase: this.ignore_case()
					},
				}
			)

			if (res.admin.data.__typename === 'ErrorPayload') {
				this.result(res.admin.data.message)
				return
			}

			if (res.admin.data.__typename === 'UpdateRedirectPayload') {
				this.result('Redirect updated successfully')
				this.data(null) // Reset data to refresh
				return
			}

			this.result('Unexpected response type')
		}
	}
}