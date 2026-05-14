namespace $.$$ {
	const request = $trip2g_graphql_request(/* GraphQL */ `
		query AdminFormSubmits($notePathId: Int64!) {
			admin {
				formSubmits(notePathId: $notePathId) {
					nodes {
						id
						noteVersionId
						formId
						ip
						status
						createdAt
						fields {
							... on AdminFormStringValue { name value }
							... on AdminFormIntValue { name value }
							... on AdminFormBoolValue { name value }
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_form_submits extends $.$trip2g_admin_form_submits {
		@$mol_mem
		data( reset?: null ) {
			const id = this.note_path_id()
			if( !id ) return []

			const res = request({ notePathId: id })
			return res.admin.formSubmits.nodes
		}

		override rows() {
			return this.data().map( ( _: any, i: number ) => this.Row( i ) )
		}

		row( i: number ) {
			return this.data()[ i ]
		}

		override row_created_at( i: number ): string {
			const ts = this.row( i ).createdAt
			return new $mol_time_moment( ts ).toString( 'DD.MM.YYYY hh:mm:ss' )
		}

		override row_ip( i: number ): string {
			return this.row( i ).ip || '-'
		}

		override row_form_id( i: number ): string {
			const id = this.row( i ).formId
			return id || '-'
		}

		override row_status( i: number ): string {
			return this.row( i ).status
		}

		override row_fields_text( i: number ): string {
			const fields: any[] = this.row( i ).fields
			if( !fields?.length ) return '-'
			return fields.map( f => `${ f.name }: ${ f.value }` ).join( ', ' )
		}
	}
}
