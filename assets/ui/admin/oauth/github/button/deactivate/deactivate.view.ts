namespace $.$$ {
	const deactivate_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminDeactivateGitHubOAuth {
			admin {
				data: deactivateGitHubOAuth {
					__typename
					... on ErrorPayload {
						message
					}
					... on DeactivateGitHubOAuthPayload {
						success
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_github_button_deactivate extends $.$trip2g_admin_oauth_github_button_deactivate {
		deactivate() {
			if( !confirm( 'Disable GitHub OAuth? Users will not be able to login via GitHub.' ) ) return

			const res = deactivate_mutation()

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			// Trigger catalog refresh
			this.$.$mol_state_arg.value( 'refresh', Date.now().toString() )
			location.reload()
		}
	}
}
