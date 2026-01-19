namespace $.$$ {
	const delete_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminDeleteGoogleOAuthCredentials($input: DeleteGoogleOAuthCredentialsInput!) {
			admin {
				data: deleteGoogleOAuthCredentials(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on DeleteGoogleOAuthCredentialsPayload {
						deletedId
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_google_delete extends $.$trip2g_admin_oauth_google_delete {
		delete() {
			const res = delete_mutation({
				input: { id: this.credentials_id() },
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			if( res.admin.data.__typename === 'DeleteGoogleOAuthCredentialsPayload' ) {
				this.$.$mol_state_arg.value( 'id', null )
				this.$.$mol_state_arg.value( 'action', null )
			}
		}
	}
}
