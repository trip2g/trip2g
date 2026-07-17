namespace $.$$ {
	const note_content_query = $trip2g_graphql_raw_request( `
		query CodeLLMDebugerDocumentNotePaths($filter: NotePathsFilter!) {
			notePaths(filter:$filter) {
				id
				content
			}
		}
	`)

	export class $trip2g_codellm_debugger_document extends $.$trip2g_codellm_debugger_document {
		@$mol_mem
		note_content() {
			const data = note_content_query( {
				filter: {
					paths: [ this.file_path() ],
				}
			} )

			return data.notePaths[ 0 ]?.content || ''
		}

		@$mol_mem
		parse_result() {
			const data = $trip2g_codellm_debugger_document_parse( {
				input: {
					content: this.note_content(),
				}
			} )

			const payload = data.parseMarkdown
			if( !( 'blocks' in payload ) ) throw new Error( payload.message )
			return payload
		}

		override step_pages(): readonly ( $mol_view )[] {
			return this.parse_result().blocks
				.map( ( block: any, idx: number ) => block.__typename === 'MarkdownCodeBlock' ? this.CodeBlock( idx ) : null )
				.filter( Boolean ) as $mol_view[]
		}

		@$mol_mem
		run_result( next?: any ): any {
			return next
		}

		run( idx: number ) {
			const blocks = this.parse_result().blocks.map( ( block: any ) => {
				const input: any = { kind: block.__typename === 'MarkdownCodeBlock' ? 'CODE' : 'PROSE', content: block.content }
				if( block.__typename === 'MarkdownCodeBlock' && block.language ) input.language = block.language
				return input
			} )
			const result = $trip2g_codellm_debugger_document_run( {
				input: {
					input: { changedFiles: [], attachedNotes: [], depth: 1 },
					maxSteps: idx+1,
					blocks,
				}
			} )

			const payload = result.runBlocks
			if( payload.__typename === 'ErrorPayload' ) throw new Error( payload.message )
			if( payload.__typename === 'BlockErrorPayload' ) {
				this.run_result( { ...payload, blockIndex: idx, error: payload.message } )
				return
			}
			this.run_result( { ...payload, blockIndex: idx } )
		}

		override block_data( idx: number ) {
			const block = this.parse_result().blocks[ idx ] as any
			if( !block ) return null
			if( block.__typename !== 'MarkdownCodeBlock' ) return block
			const parsedBlocks = this.parse_result().blocks as readonly any[]
			const codeIndex = parsedBlocks
				.slice( 0, idx )
				.filter( ( item: any ) => item.__typename === 'MarkdownCodeBlock' ).length
			const runResult = this.run_result()
			const results = runResult?.results || []
			const result = results.find( ( item: any ) => item.index === codeIndex )
			const lastCodeIndex = typeof runResult?.blockIndex === 'number'
				? parsedBlocks
					.slice( 0, runResult.blockIndex + 1 )
					.filter( ( item: any ) => item.__typename === 'MarkdownCodeBlock' ).length - 1
				: -1
			return {
				...block,
				block_title: `${ codeIndex + 1 }. ${ block.language || 'code' }`,
				prose: parsedBlocks
					.slice( this.prose_start( parsedBlocks, idx ), idx )
					.filter( ( item: any ) => item.__typename === 'MarkdownProseBlock' )
					.map( ( item: any ) => item.html )
					.join( '' ),
				stdout: result?.stdout || '',
				stderr: result?.stderr || '',
				exitCode: result?.exitCode ?? 0,
				durationMs: result?.durationMs || 0,
				maxRssBytes: result?.maxRssBytes || 0,
				output: codeIndex === lastCodeIndex ? runResult?.output || '' : '',
				error: idx === runResult?.blockIndex ? runResult?.error || '' : '',
				run: () => this.run( idx ),
			}
		}

		prose_start( blocks: readonly any[], codeIndex: number ) {
			for( let index = codeIndex - 1; index >= 0; index-- ) {
				if( blocks[ index ].__typename === 'MarkdownCodeBlock' ) return index + 1
			}
			return 0
		}

		override raw_content(): string {
			return this.note_content()
		}
	}

	export class $trip2g_codellm_debugger_document_prose_block extends $.$trip2g_codellm_debugger_document_prose_block {
		override content() {
			// @ts-ignore
			return this.data()?.html || ''
		}
	}

	export class $trip2g_codellm_debugger_document_code_block extends $.$trip2g_codellm_debugger_document_code_block {
		override block_title() {
			return ( this.data() as any )?.block_title || ''
		}

		override prose_content() {
			return ( this.data() as any )?.prose || ''
		}

		override content() {
			// @ts-ignore
			return this.data()?.content || ''
		}

		override handle_run() {
			const data = this.data() as any
			data?.run?.()
		}

		override exit_code() {
			return `Exit ${ ( this.data() as any )?.exitCode ?? 0 }`
		}

		override stats_content() {
			const data = this.data() as any
			const seconds = ( ( data?.durationMs || 0 ) / 1000 ).toFixed( 0 )
			const megabytes = ( ( data?.maxRssBytes || 0 ) / 1024 / 1024 ).toFixed( 2 )
			return `${ seconds }s · ${ megabytes } MB`
		}

		override stdout_content() {
			const data = this.data() as any
		return data?.output || data?.stdout || ''
		}

		override stderr_content() {
			const data = this.data() as any
			return data?.stderr || ''
		}

		override error_content() {
			return ( this.data() as any )?.error || ''
		}
	}
}
