namespace $.$$ {
	const data_request = $trip2g_graphql_request(
		`
			query AdminShowHtmlInjection($id: Int64!) {
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
		`
	)

	export class $trip2g_admin_htmlinjection_show extends $.$trip2g_admin_htmlinjection_show {
		action() {
			return this.$.$mol_state_arg.value( 'action' ) || 'view'
		}

		@$mol_mem
		data( reset?: null ) {
			const res = data_request({
				id: this.htmlinjection_id()
			})

			const injection = res.admin.htmlInjection
			if( !injection ) {
				throw new Error( 'Html Injection not found' )
			}

			return injection
		}

		override body() {
			if( this.action() === 'update' ) {
				return [ this.UpdateForm() ]
			}

			return [ this.InjectionDetails() ]
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
	}
}
