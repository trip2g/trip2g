namespace $.$$ {
	const diff_request = $trip2g_graphql_request(/* GraphQL */ `
		query EditorNoteVersionDiff($filter: NoteVersionDiffFilter!) {
			admin {
				noteVersionDiff(filter: $filter) {
					unified
					word
					addedLines
					removedLines
					changedWords
				}
			}
		}
	`)

	export class $trip2g_editor_diff extends $.$trip2g_editor_diff {
		@$mol_mem
		diff_result() {
			const from = this.from_version_id()
			const to = this.to_version_id()
			if (!from || !to) return null
			const res = diff_request({ filter: { fromVersionId: from, toVersionId: to } })
			return res.admin.noteVersionDiff ?? null
		}

		override stats_rows(): readonly $mol_view[] {
			const d = this.diff_result()
			if (!d) return []
			const label = (text: string) => {
				const v = new $mol_view()
				v.sub = () => [text]
				return v
			}
			return [
				label(`+${d.addedLines} lines  -${d.removedLines} lines  ~${d.changedWords} words`),
			]
		}

		override unified_lines(): readonly $mol_view[] {
			const d = this.diff_result()
			if (!d || !d.unified) return []
			return d.unified.split('\n').map(line => {
				const v = new $mol_view()
				const cls = $trip2g_editor_diff_line_class(line)
				v.attr = () => cls ? { class: cls } : {}
				v.sub = () => [line || ' ']
				return v
			})
		}
	}
}
