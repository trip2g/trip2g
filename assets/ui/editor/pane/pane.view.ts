namespace $.$$ {
	const content_request = $trip2g_graphql_request(/* GraphQL */ `
		query EditorNoteContent($filter: NotePathsFilter) {
			notePaths(filter: $filter) {
				id
				value
				content
			}
		}
	`)

	const save_mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation EditorPushNotes($input: PushNotesInput!) {
			pushNotes(input: $input) {
				__typename
				... on ErrorPayload {
					message
				}
				... on PushNotesPayload {
					updated {
						path
						url
					}
				}
			}
		}
	`)

	export class $trip2g_editor_pane extends $.$trip2g_editor_pane {
		@$mol_mem
		override path(next?: string): string {
			if (next !== undefined) {
				const w = this.$.$mol_dom_context as unknown as Window
				if (w.parent && w.parent !== w) {
					w.parent.postMessage({ type: 'trip2g_editor_path', path: next }, '*')
				}
				return next
			}
			return $trip2g_settings.note_path()
		}

		@$mol_mem_key
		loaded_note_path(path: string): { id: any; content: string } | null {
			if (!path) return null
			const res = content_request({ filter: { paths: [path] } })
			const np = res.notePaths[0]
			return np ? { id: np.id, content: np.content } : null
		}

		@$mol_mem_key
		override loaded_content(path: string): string {
			return this.loaded_note_path(path)?.content ?? ''
		}

		note_path_id(): any {
			return this.loaded_note_path(this.path())?.id ?? null
		}

		editor_key(suffix: string): string {
			return `trip2g_editor:${suffix}`
		}

		override changed_paths(next?: string[]): string[] {
			return this.$.$mol_state_local.value(this.editor_key('change:pathes'), next) ?? []
		}

		change(path: string, next?: string | null): string | null {
			return this.$.$mol_state_local.value(this.editor_key(`change:path:${path}`), next) ?? null
		}

		override content(next?: string): string {
			const path = this.path()
			if (next !== undefined) {
				if (path) {
					this.change(path, next)
					const paths = this.changed_paths()
					if (!paths.includes(path)) this.changed_paths([...paths, path])
				}
				return next
			}
			if (!path) return ''
			const stored = this.change(path)
			return stored !== null ? stored : this.loaded_content(path)
		}

		override file_title(): string {
			return this.path() || 'Editor'
		}

		override editor_body(): readonly $mol_view[] {
			return this.path() ? [this.ContentTextarea()] : [this.Placeholder()]
		}

		override has_content(): boolean {
			return this.content().length > 0
		}

		override dirty(): boolean {
			return this.changed_paths().length > 0
		}

		override save_text(): string {
			return super.save_text().replace('{0}', String(this.changed_paths().length))
		}

		download_name(): string {
			const path = this.path()
			const name = path ? path.split('/').at(-1) ?? 'note.md' : 'note.md'
			return name.endsWith('.md') ? name : `${name}.md`
		}

		override download(next?: Event): null {
			if (next !== undefined) {
				const blob = new Blob([this.content()], { type: 'text/markdown' })
				const uri = URL.createObjectURL(blob)
				const a = this.$.$mol_dom_context.document.createElement('a')
				a.href = uri
				a.download = this.download_name()
				a.click()
				URL.revokeObjectURL(uri)
			}
			return null
		}

		save_selected(paths: readonly string[]): void {
			const updates = paths
				.map(p => ({ path: p, content: this.change(p) }))
				.filter((u): u is { path: string; content: string } => u.content !== null)
			if (updates.length === 0) return
			const res = save_mutate({ input: { updates } })
			if (res.pushNotes.__typename === 'ErrorPayload') {
				throw new Error(res.pushNotes.message)
			}
			this.changed_paths(this.changed_paths().filter(p => !paths.includes(p)))
			for (const p of paths) this.change(p, null)
		}

		override save_confirm(next?: Event): null {
			if (next !== undefined) {
				this.save_selected(this.SaveList().selected())
				this.toggle_right_sidebar('save')
			}
			return null
		}

		right_sidebar_widget(): $mol_view | null {
			const w = this.right_sidebar()
			return w ? this.right_widgets()[w] : null
		}

		toggle_right_sidebar(widget: string) {
			const v = this.right_sidebar()
			return this.right_sidebar(v === widget ? '' : widget)
		}

		@$mol_mem
		right_widgets(): { [key: string]: $mol_view } {
			return {
				versions: this.Versions(),
				save: this.SaveList(),
			}
		}

		override handle_versions_click() {
			this.toggle_right_sidebar('versions')
		}

		override handle_save_click() {
			this.toggle_right_sidebar('save')
		}

		override close(): null {
			const w = this.$.$mol_dom_context as unknown as Window
			if (w.parent && w.parent !== w) {
				w.parent.postMessage('trip2g_editor_close', '*')
			}
			return null
		}
	}
}
