namespace $.$$ {
	export class $trip2g_admin_htmlinjection_show extends $.$trip2g_admin_htmlinjection_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request(
				`
					query AdminShowHTMLInjection($id: Int64!) {
						admin {
							htmlInjection(id: $id) {
								id
								createdAt
								activeFrom
								activeTo
								description
								position
								placement
								content
							}
						}
					}
				`,
				{
					id: this.htmlinjection_id()
				}
			)

			const injection = res.admin.htmlInjection
			if( !injection ) {
				throw new Error( 'HTML Injection not found' )
			}

			return injection
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return [ this.InjectionDetails(), this.DeleteResult() ]
		}

		injection_id(): string {
			return this.data().id.toString()
		}

		injection_description(): string {
			return this.data().description || '-'
		}

		injection_placement(): string {
			return this.data().placement
		}

		injection_position(): string {
			return this.data().position.toString()
		}

		injection_content(): string {
			return this.data().content
		}

		injection_active_from(): string {
			const activeFrom = this.data().activeFrom
			if (!activeFrom) return '-'
			const m = new $mol_time_moment( activeFrom )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		injection_active_to(): string {
			const activeTo = this.data().activeTo
			if (!activeTo) return '-'
			const m = new $mol_time_moment( activeTo )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		injection_created_at(): string {
			const m = new $mol_time_moment( this.data().createdAt )
			return m.toString( 'YYYY-MM-DD HH:mm' )
		}

		delete() {
			const res = $trip2g_graphql_request(
				`
					mutation AdminDeleteHTMLInjectionMutation($input: DeleteHTMLInjectionInput!) {
						admin {
							data: deleteHTMLInjection(input: $input) {
								... on DeleteHTMLInjectionPayload {
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
						id: this.htmlinjection_id()
					},
				}
			)

			if( res.admin.data.__typename === 'ErrorPayload' ) {
				this.delete_result( res.admin.data.message )
				return
			}

			if( res.admin.data.__typename === 'DeleteHTMLInjectionPayload' ) {
				this.delete_result( 'HTML Injection deleted successfully' )
				// Navigate back to catalog
				this.$.$mol_state_arg.value( 'id', '' )
				return
			}

			this.delete_result( 'Unexpected response type' )
		}
	}
}