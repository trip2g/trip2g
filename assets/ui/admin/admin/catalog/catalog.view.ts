namespace $.$$ {
	export class $trip2g_admin_admin_catalog extends $.$trip2g_admin_admin_catalog {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_admin_admin_catalog_list()

			return $trip2g_graphql_make_map( res.admin.allAdmins.nodes )
		}

		@$mol_mem
		spreads(): any {
			return {
				add: this.CreateForm(),
				...this.data().mapKeys( key => this.ShowPage( key ) )
			}
		}

		@$mol_mem
		override spread_ids_filtered() {
			return this.spread_ids().filter( id => id !== 'add' )
		}

		row( id: any ) {
			return this.data().get( id )
		}

		override row_id( id: any ): number {
			return this.row( id ).id
		}

		override row_id_string( id: any ): string {
			return this.row( id ).id.toString()
		}

		override row_user_email( id: any ): string {
			return this.row( id ).user?.email || '???'
		}

		override row_granted_at( id: any ): string {
			return this.row( id ).grantedAt
		}
	}
}
