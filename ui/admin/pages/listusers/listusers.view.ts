namespace $.$$ {
	type Response = {
		rows: {
			id: number;
			email: string;
		}[];
	}

	export class $trip2g_admin_pages_listusers extends $.$trip2g_admin_pages_listusers {
		@$mol_mem
		data() {
			const data = this.$.$mol_fetch.json( '/api/admin/listusers' ) as Response;
			const map: { [ id: string ]: Response['rows'][0] } = {};

			data.rows.forEach( ( row ) => {
				map[ row.id ] = row
			})

			return {
				map,
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

		row_email( id: any ): string {
			return this.data().map[ id ].email;
		}
	}
}
