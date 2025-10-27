namespace $.$$ {
	const mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {
			admin {
				payload: makeReleaseLive(input:$input) {
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
	`)

	export class $trip2g_admin_release_button_makelive extends $.$trip2g_admin_release_button_makelive {
		override handle_click() {
			const res = mutate({
				input: {
					id: this.release_id(),
				},
			})

			if( res.admin.payload.__typename === 'ErrorPayload' ) {
				this.result( res.admin.payload.message )
				return
			}

			if( res.admin.payload.__typename === 'MakeReleaseLivePayload' ) {
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