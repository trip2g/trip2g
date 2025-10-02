namespace $.$$ {
	export class $trip2g_admin_htmlinjection_create extends $.$trip2g_admin_htmlinjection_create {
		override body() {
			if( this.injection_id_string() !== '' ) {
				return [ this.InjectionView() ]
			}

			return super.body()
		}

		override description_bid(): string {
			// Description is optional
			return ''
		}

		override placement_bid(): string {
			const placement = this.placement()
			if( !placement ) {
				return 'Placement is required'
			}
			return ''
		}

		override position_bid(): string {
			const position = this.position()
			if( position === null || position === undefined ) {
				return 'Position is required'
			}
			if( position < 0 ) {
				return 'Position must be non-negative'
			}
			return ''
		}

		override content_bid(): string {
			const content = this.content()
			if( !content.trim() ) {
				return 'Content is required'
			}
			return ''
		}

		override active_from_moment(next?: $mol_time_moment | null) {
			if (next) {
				next = new $mol_time_moment().merge(next);
			}

			return super.active_from_moment(next)
		}

		override active_to_moment(next?: $mol_time_moment | null) {
			if (next) {
				next = new $mol_time_moment().merge(next);
			}

			return super.active_to_moment(next)
		}

		submit() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminCreateHTMLInjectionMutation($input: CreateHTMLInjectionInput!) {
						admin {
							data: createHTMLInjection(input: $input) {
								... on CreateHTMLInjectionPayload {
									htmlInjection {
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
						description: this.description() || null,
						placement: this.placement(),
						position: this.position(),
						content: this.content(),
						activeFrom: $trip2g_moment_toserver(this.active_from_moment()),
						activeTo: $trip2g_moment_toserver(this.active_to_moment()),
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'CreateHTMLInjectionPayload' ) {
				this.injection_id_string( res.admin.data.htmlInjection.id.toString() )
				return
			}

			this.result( 'Unexpected response type' )
		}
	}
}
