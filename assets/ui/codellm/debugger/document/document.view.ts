namespace $.$$ {
	const note_content_query = $trip2g_graphql_raw_request( `
		query CodeLLMDebugerDocumentNotePaths($filter: NotePathsFilter!) {
			notePaths(filter:$filter) {
				id
				content
			}
		}
	`)

	const parse_query = $trip2g_codellm_graphql_raw_request( `
		query ParseQueryDocumentQuery($input: ParseMarkdownInput!) {
			parseMarkdown(input: $input) {
				... on ErrorPayload {
					message
				}
				... on ParseMarkdownPayload {
					blocks {
						__typename

						... on MarkdownCodeBlock {
							index
							language
							content
						}

						... on MarkdownProseBlock {
							index
							content
							index
							html
						}
					}
				}
			}
		}
	`)

	const run_query = $trip2g_codellm_graphql_raw_request( `
		mutation RunBlocks($input: RunBlocksInput!) {
			runBlocks(input: $input) {
				__typename
				... on RunBlocksPayload { output pipes { index content } }
				... on ErrorPayload { message }
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
			const data = parse_query( {
				input: {
					content: this.note_content(),
				}
			} )

			return data.parseMarkdown
		}

		override blocks(): readonly ( $mol_view )[] {
			return this.parse_result().blocks.map( ( block: any, idx: number ) => {
				if( block.__typename === 'MarkdownCodeBlock' ) {
					return this.CodeBlock( idx )
				}

				if( block.__typename === 'MarkdownProseBlock' ) {
					return this.ProseBlock( idx )
				}

				throw new Error( `Unknown block type: ${ block.__typename }` )
			} )
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
			const result = run_query( {
				input: {
					input: { changedFiles: [], attachedNotes: [], depth: 1 },
					maxSteps: idx+1,
					blocks,
				}
			}, {
				resetCache: true,
			} )

			const payload = result.runBlocks
			if( payload.__typename === 'ErrorPayload' ) throw new Error( payload.message )
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
			const pipes = runResult?.pipes || []
			const lastCodeIndex = typeof runResult?.blockIndex === 'number'
				? parsedBlocks
					.slice( 0, runResult.blockIndex + 1 )
					.filter( ( item: any ) => item.__typename === 'MarkdownCodeBlock' ).length - 1
				: -1
			return {
				...block,
				pipe: pipes.find( ( pipe: any ) => pipe.index === codeIndex )?.content || '',
				output: codeIndex === lastCodeIndex ? runResult?.output || '' : '',
				run: () => this.run( idx ),
			}
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
		override content() {
			// @ts-ignore
			return this.data()?.content || ''
		}

		override handle_run() {
			const data = this.data() as any
			data?.run?.()
		}

		override pipe_content() {
			const data = this.data() as any
			return data?.output || data?.pipe || ''
		}
	}
}
