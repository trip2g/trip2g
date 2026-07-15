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

	export class $trip2g_codellm_debugger_old extends $.$trip2g_codellm_debugger {
		@$mol_mem
		files() { return [ 'examples/hello.md', 'examples/jsonl.md', 'scratch.md' ] }

		@$mol_mem
		file( next?: string ) {
			if( next !== undefined ) return next
			return this.files()[ 0 ]
		}

		@$mol_mem
		markdown( next?: string ) {
			if( next !== undefined ) return next
			return '```bash\necho \'{"hello":{}}\'\n```\n\n```bash\njq .hello\n```\n'
		}

		@$mol_mem
		fleet_input( next?: string ) {
			if( next !== undefined ) return next
			return JSON.stringify( { changedFiles: [], attachedNotes: [], depth: 1 }, null, 2 )
		}

		@$mol_mem
		max_steps( next?: string ) { return next === undefined ? '0' : next }

		blocks(): Block[] {
			const out: Block[] = []
			const re = /```([^\n]*)\n([\s\S]*?)```/g
			let match: RegExpExecArray | null
			while( ( match = re.exec( this.markdown() ) ) ) {
				out.push( { kind: 'CODE', language: match[ 1 ].trim() || 'text', content: match[ 2 ] } )
			}
			return out
		}

		file_rows() { return this.files().map( ( name, i ) => this.File( i ) ) }
		file_title( i: number ) { return this.files()[ i ] }
		file_click( i: number ) { return () => this.file( this.files()[ i ] ) }
		block_rows() {
			return this.blocks().map( ( _, i ) => {
				const view = this.Block( i ) as any
				view.__block_index = i
				return view
			} )
		}
		block_language( i: number ) { return this.blocks()[ i ].language || 'text' }
		block_code( i: number, next?: string ) { return next === undefined ? this.blocks()[ i ].content : next }
		block_number( i: number ) { return `#${ i + 1 }` }

		async execute( max: number ) {
			const parsed = JSON.parse( this.fleet_input() || '{}' )
			const res = await run_request( { input: { input: parsed, maxSteps: max, blocks: this.blocks().map( b => ( { kind: b.kind, language: b.language, content: b.content } ) ) } } )
			const payload = res.runBlocks
			if( payload.__typename === 'ErrorPayload' ) throw new Error( payload.message )
			this.result( { output: payload.output || '', pipes: payload.pipes || [] } )
		}

		@$mol_mem
		result( next?: any ) { return next === undefined ? { output: '', pipes: [] } : next }
		run() { this.execute( Number( this.max_steps() ) || this.blocks().length ) }
		run_step() { this.execute( 1 ) }
		status() { const r = this.result(); return r.output ? `Output: ${ r.output }` : 'Ready' }
		pipe_rows() { return this.result().pipes.map( ( p: any ) => { const v = new this.$.$mol_view(); v.sub = () => [ `pipe ${ p.index }: ${ p.content || '(empty)' }` ]; return v } ) }
	}

	export class $trip2g_codellm_debugger_block extends $.$trip2g_codellm_debugger_block {
		language() { return ( this as any ).owner().block_language( this.index() ) }
		code( next?: string ) { return ( this as any ).owner().block_code( this.index(), next ) }
		number() { return ( this as any ).owner().block_number( this.index() ) }
		output_view() { return [] }
	}
}
