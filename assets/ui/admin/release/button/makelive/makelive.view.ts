namespace $.$$ {
	export class $trip2g_admin_release_button_makelive extends $.$trip2g_admin_release_button_makelive {
		override handle_click() {
			const res = $trip2g_graphql_request( `
					mutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {
						admin {
							data: makeReleaseLive(input:$input) {
								... on ErrorPayload {
									message
								}
								... on MakeReleaseLivePayload {
									release {
										id
									}
								}
							}
						}
					}
				`,
				{
					input: {
						id: this.release_id(),
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'MakeReleaseLivePayload' ) {
				this.result( 'Release is now live' )
				return
			}

			this.result( 'Unexpected response type' )
		}

		override sub() {
			if( this.is_live() ) {
				return []
			}

			return super.sub()
		}
	}
}