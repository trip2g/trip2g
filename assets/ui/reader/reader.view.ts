namespace $.$$ {
	export class $trip2g_reader extends $.$trip2g_reader {
		@$mol_mem
		data( reset?: null ) {
			const res = $trip2g_graphql_request( `
				query ReaderQuery($input: NoteInput!) {
					note(input: $input) {
						title
						html
					}
				}
			`, {
				input: {
					path: "/ponedeljnik_9_iyunya_2025",
					referer: "",
				},
			} )

			return res.note
		}

		title() {
			return this.data()?.title || '???'
		}
	}
}