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
						index
						kind
						language
						content
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

		override test(): string {
			console.log(this.parse_result())
			return ''
		}
	}
}
