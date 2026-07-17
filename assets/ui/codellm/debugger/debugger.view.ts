namespace $.$$ {
	type Block = { kind: 'CODE' | 'PROSE', language?: string, content: string }

	export class $trip2g_codellm_debugger extends $.$trip2g_codellm_debugger {
		@$mol_mem
		documents() {
			const req = $trip2g_codellm_debugger_note_paths( {
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
