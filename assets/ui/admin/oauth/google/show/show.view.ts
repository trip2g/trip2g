namespace $.$$ {
	const query = $trip2g_graphql_request(/* GraphQL */ `
		query AdminGoogleOAuthCredentialsById($id: Int!) {
			admin {
				googleOAuthCredentials(id: $id) {
					id
					name
					clientId
					active
					createdAt
					createdBy { id email }
				}
			}
		}
	`)

	const set_active_mutation = $trip2g_graphql_request(/* GraphQL */ `
		mutation AdminSetActiveGoogleOAuthCredentials($input: SetActiveGoogleOAuthCredentialsInput!) {
			admin {
				data: setActiveGoogleOAuthCredentials(input: $input) {
					__typename
					... on ErrorPayload {
						message
					}
					... on SetActiveGoogleOAuthCredentialsPayload {
						credentials {
							id
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_oauth_google_show extends $.$trip2g_admin_oauth_google_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data( reset?: null ) {
			return query({ id: this.credentials_id() }).admin.googleOAuthCredentials
		}

		override body() {
			if( this.action() === 'delete' ) {
				return [ this.DeleteForm() ]
			}

			return super.body()
		}

		credentials_id_string() {
			return String( this.data().id )
		}

		credentials_name() {
			return this.data().name
		}

		credentials_client_id() {
			return this.data().clientId
		}

		credentials_active() {
			return this.data().active ? 'Yes' : 'No'
		}

		credentials_created_at() {
			return this.data().createdAt
		}

		credentials_created_by() {
			const user = this.data().createdBy
			return user?.email || String( user?.id || '' )
		}

		is_active() {
			return this.data().active
		}

		set_active_title() {
			return this.data().active ? 'Active' : 'Set Active'
		}

		set_active_click() {
			if( this.data().active ) return

			const res = set_active_mutation({
				input: { id: this.credentials_id() },
			})

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				throw new Error( res.admin.data.message )
			}

			this.data( null )
		}
	}
}
