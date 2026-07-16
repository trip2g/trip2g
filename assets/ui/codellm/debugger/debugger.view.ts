namespace $.$$ {
	const run_query = /* GraphQL */ `
		mutation RunBlocks($input: RunBlocksInput!) {
			runBlocks(input: $input) {
				... on RunBlocksPayload { output pipes { index content } }
				... on ErrorPayload { message }
			}
		}
	`
	const run_request = async ( variables: any ) => {
		const response = await fetch( '/_system/codellm/graphql', {
			method: 'POST', credentials: 'same-origin',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify( { query: run_query, variables } ),
		} )
		const body = await response.json()
		if( body.errors?.length ) throw new Error( body.errors[ 0 ].message )
		return body.data
	}

	const note_paths_query = $trip2g_graphql_raw_request( `
		query CodeLLMDebugerNotePaths($filter: NotePathsFilter!) {
			notePaths(filter:$filter) {
				id
				value
				latestNoteView {
					title
				}
			}
		}
	`)

	type Block = { kind: 'CODE' | 'PROSE', language?: string, content: string }

	export class $trip2g_codellm_debugger extends $.$trip2g_codellm_debugger {
		@$mol_mem
		documents() {
			const req = note_paths_query( {
				filter: {
					frontmatter: [ { key: "fleet_id", equals: "codellm" } ],
				}
			} )

			const res = {
				map: {} as Record<string, any>,
				ids: [] as string[],
			}

			req.notePaths.forEach( ( doc: any ) => {
				res.map[ doc.id ] = doc
				res.ids.push( doc.id )
			} )

			return res
		}

		@$mol_mem
		override spreads(): Record<string, any> {
			const files: Record<string, any> = {}

			this.documents().ids.forEach( ( id: any ) => {
				files[ id ] = this.Page( id )
			} )

			return files
		}

		override file_path( id: any ): string {
			return this.documents().map[id].value
		}

		override file_label( id: any ): string {
			const doc = this.documents().map[id]
			return doc.latestNoteView?.title || doc.value
		}
	}
}
