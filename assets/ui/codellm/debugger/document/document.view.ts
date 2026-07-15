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
			const data = parse_query({
				input: {
					content: this.note_content(),
				}
			})

			return data.parseMarkdown;
		}

		override blocks(): readonly ( $mol_view )[] {
			return this.parse_result().blocks.map( ( block: any, idx: number ) => {
				if (block.__typename === 'MarkdownCodeBlock') {
					return this.CodeBlock(idx)
				}

				if (block.__typename === 'MarkdownProseBlock') {
					return this.ProseBlock(idx)
				}

				throw new Error(`Unknown block type: ${block.__typename}`)
			})
		}

		override block_data( idx: number ) {
			return this.parse_result().blocks[idx]
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
	}
}
