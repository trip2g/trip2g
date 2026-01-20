namespace $.$$ {
	const deactivate_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminDeactivateGoogleOAuth {
			admin {
				data: deactivateGoogleOAuth {
					__typename
					... on ErrorPayload {
						message
					}
					... on DeactivateGoogleOAuthPayload {
						success
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_google_disableall extends $.$trip2g_admin_oauth_google_disableall {
		disable() {
			const res = deactivate_mutation()

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			this.$.$mol_state_arg.value( 'id', null )
		}
	}
}
