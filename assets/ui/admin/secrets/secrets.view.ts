namespace $.$$ {
	const keysQuery = $trip2g_graphql_request(/* GraphQL */`
		query AdminSecretKeys($filter: SecretKeysFilter) {
			admin {
				secretKeys(filter: $filter)
			}
		}
	`)

	const setMutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminSetSecret($input: SetSecretInput!) {
			admin {
				setSecret(input: $input) {
					key
				}
			}
		}
	`)

	const deleteMutate = $trip2g_graphql_request(/* GraphQL */`
		mutation AdminDeleteSecret($id: String!) {
			admin {
				deleteSecret(id: $id) {
					id
				}
			}
		}
	`)

	export class $trip2g_admin_secrets extends $.$trip2g_admin_secrets {
		@$mol_mem
		keys( reset?: null ): string[] {
			const prefix = this.key_prefix()
			if( !prefix ) return []
			return keysQuery({ filter: { idPrefix: prefix } }).admin.secretKeys
		}

		key_rows() {
			return this.keys().map( ( _, i ) => this.KeyRow( i ) )
		}

		key_name( index: number ) {
			const prefix = this.key_prefix()
			const key = this.keys()[ index ] ?? ''
			return key.startsWith( prefix ) ? key.slice( prefix.length ) : key
		}

		@$mol_mem_key
		key_new_value( index: number, next?: string ): string { return next ?? '' }

		key_row_sub( index: number ) {
			const action = this.key_new_value( index )
				? this.KeySave( index )
				: this.KeyDelete( index )
			return [ this.KeyName( index ), this.KeyValueInput( index ), action ]
		}

		key_save( index: number, next?: null ) {
			if( next === undefined ) return null
			const key = this.keys()[ index ]
			if( !key ) return null
			setMutate({ input: { key, value: this.key_new_value( index ) } })
			this.key_new_value( index, '' )
			this.keys( null )
			return null
		}

		key_delete( index: number, next?: null ) {
			if( next === undefined ) return null
			const key = this.keys()[ index ]
			if( !key ) return null
			deleteMutate({ id: key })
			this.keys( null )
			return null
		}

		@$mol_mem
		new_name( next?: string ): string { return next ?? '' }

		@$mol_mem
		new_value( next?: string ): string { return next ?? '' }

		add_secret( next?: null ) {
			if( next === undefined ) return null
			const name = this.new_name().trim()
			const value = this.new_value().trim()
			if( !name || !value ) {
				this.add_result( 'Name and value are required' )
				return null
			}
			const key = this.key_prefix() + name
			setMutate({ input: { key, value } })
			this.new_name( '' )
			this.new_value( '' )
			this.add_result( 'Secret added' )
			this.keys( null )
			return null
		}
	}
}
