namespace $.$$ {
	export class $trip2g_admin_pages_listusersubgraphaccesses extends $.$trip2g_admin_pages_listusersubgraphaccesses {
		@$mol_mem
		data() {
			const data = this.$.$mol_fetch.json( '/api/admin/listusersubgraphacesses' ) as Response;
			const map: { [ id: string ]: Response['rows'][0] } = {};
			const userMap: { [ id: string ]: Response['users'][0] } = {};
			const subgraphMap: { [ id: string ]: Response['subgraphs'][0] } = {};

			data.rows.forEach( ( row ) => {
				map[ row.id ] = row
			})

			data.users.forEach( ( row ) => {
				userMap[ row.id ] = row
			})

			data.subgraphs.forEach( ( row ) => {
				subgraphMap[ row.id ] = row
			})

			return {
				map,
				userMap,
				subgraphMap,
				ids: Object.keys( map ),
			}
		}

		@$mol_mem
		spreads(): any {
			const pages: { [ id: string ]: any } = {};

			this.data().ids.forEach( (id) => {
				pages[id] = this.Content(id);
			});

			return pages;
		}

		row_id( id: any ): string {
			return this.data().map[ id ].id.toString();
		}

		row_subgraph_name( id: any ): string {
			const subgraphId = this.data().map[ id ].subgraph_id;
			return this.data().subgraphMap[ subgraphId ].name;
		}
	}

	export interface Response {
		rows: Row[]
		subgraphs: Subgraph[]
		users: User[]
		success: boolean
		errors: any
	  }
	  
	  export interface Row {
		id: number
		user_id: number
		subgraph_id: number
		purchase_id: PurchaseId
		created_at: string
		expires_at: ExpiresAt
		revoke_id: RevokeId
		user_email: string
		subgraph_name: string
	  }
	  
	  export interface PurchaseId {
		int64: number
		valid: boolean
	  }
	  
	  export interface ExpiresAt {
		time: string
		valid: boolean
	  }
	  
	  export interface RevokeId {
		int64: number
		valid: boolean
	  }
	  
	  export interface Subgraph {
		id: number
		name: string
		color: Color
		created_at: string
	  }
	  
	  export interface Color {
		string: string
		valid: boolean
	  }
	  
	  export interface User {
		id: number
		email: string
		created_at: string
		last_signin_code_sent_at: LastSigninCodeSentAt
	  }
	  
	  export interface LastSigninCodeSentAt {
		time: string
		valid: boolean
	  }
}
