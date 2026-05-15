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
							__typename
							... on AdminFormStringValue { name sv: value }
							... on AdminFormIntValue { name iv: value }
							... on AdminFormBoolValue { name bv: value }
						}
					}
				}
			}
		}
	`)

	export class $trip2g_admin_formsubmits_catalog extends $.$trip2g_admin_formsubmits_catalog {
		@$mol_mem
		data( reset?: null ) {
			const id = this.note_path_id()
			if( !id ) return []

			const res = request({ notePathId: id })
			return res.admin.formSubmits.nodes
		}

		override rows() {
			console.log(this.data(), this.note_path_id())
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

		@$mol_mem
		all_keys() {
			const s = new Set<string>()

			this.data().forEach((row: any) => {
				const fields: any[] = row.fields
				if( !fields?.length ) return
				fields.forEach(f => s.add(f.name))
			})

			return Array.from(s)
		}

		@$mol_mem_key
		override columns( id: any ): readonly ( $mol_labeler )[] {
			const keys = this.all_keys()
			return [
				...super.columns( id ),
				...keys.map( key => this.CustomColumn( `${id}_${key}` ) ),
			]
		}

		override row_fields_key( i: any ): string {
			const [, name] = String( i ).split('_', 2)
			return name
		}

		override row_fields_value( parent_key: any ): string {
			const [i, name] = String( parent_key ).split('_', 2)
			const field = this.row( parseInt( i ) ).fields.find((f: any) => f.name === name)
			if (!field) return '-'

			switch(field.__typename) {
				case 'AdminFormStringValue': return field.sv
				case 'AdminFormIntValue': return String(field.iv)
				case 'AdminFormBoolValue': return field.bv ? 'true' : 'false'
			}

			return '-'
		}
	}
}
