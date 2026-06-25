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

	const resolve_request = $trip2g_graphql_request(/* GraphQL */ `
		query ResolveWikilinks($filter: ResolveWikilinksFilter!) {
			resolveWikilinks(filter: $filter) {
				link
				path
				url
			}
		}
	`)

	const history_request = $trip2g_graphql_request(/* GraphQL */ `
		query EditorNoteVersionsForDiff($filter: AdminNoteVersionHistoryFilter!) {
			admin {
				noteVersionHistory(filter: $filter) {
					nodes {
						versionId
						version
					}
				}
			}
		}
	`)

	const EDITOR_CHANGES_QUERY = /* GraphQL */ `
		subscription EditorNoteChanges($filter: NoteChangesFilter!) {
			noteChanges(filter: $filter) {
				changes {
					__typename
					... on NoteUpsertEvent {
						path
						pathId
						versionId
					}
					... on NoteHideEvent {
						path
					}
				}
			}
		}
	`

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

		// Per-path reload counter: incrementing it forces loaded_note_path to re-fetch.
		@$mol_mem_key
		reload_counter(path: string, next?: number): number {
			return next ?? 0
		}

		@$mol_mem_key
		loaded_note_path(path: string): { id: any; content: string } | null {
			if (!path) return null
			this.reload_counter(path) // reactive dependency
			const res = content_request({ filter: { paths: [path] } })
			const np = res.notePaths[0]
			return np ? { id: np.id, content: np.content } : null
		}

		@$mol_mem_key
		loaded_content(path: string): string {
			return this.loaded_note_path(path)?.content ?? ''
		}

		note_path_id(): any {
			return this.loaded_note_path(this.path())?.id ?? null
		}

		wikilink_at(text: string, offset: number): string | null {
			const re = /\[\[([^\]]+)\]\]/g
			let m: RegExpExecArray | null
			while ((m = re.exec(text)) !== null) {
				if (m.index <= offset && offset <= m.index + m[0].length) {
					return m[1].split('|')[0].split('#')[0].trim()
				}
			}
			return null
		}

		override handle_content_click(next?: MouseEvent | null): null {
			if (next?.ctrlKey) {
				const pos = document.caretPositionFromPoint(next.clientX, next.clientY)
				if (pos?.offsetNode?.nodeType === Node.TEXT_NODE) {
					const link = this.wikilink_at(this.content(), pos.offset)
					if (link) {
						const pathId = this.note_path_id()
						if (pathId !== null) {
							const res = resolve_request({ filter: { notePathId: pathId, links: [link] } })
							const resolved = res.resolveWikilinks[0]
							if (resolved?.path) this.path(resolved.path)
						}
					}
				}
			}
			return null
		}

		override handle_content_hover(next?: PointerEvent | null): null {
			if (next) {
				const ta = next.target as HTMLTextAreaElement
				if (next.ctrlKey) {
					const pos = document.caretPositionFromPoint(next.clientX, next.clientY)
					const link = pos?.offsetNode?.nodeType === Node.TEXT_NODE
						? this.wikilink_at(this.content(), pos.offset)
						: null
					ta.style.cursor = link ? 'pointer' : ''
				} else {
					ta.style.cursor = ''
				}
			}
			return null
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
			const views: $mol_view[] = []
			if (this.pending_external_update()) views.push(this.UpdateBanner())
			views.push(this.path() ? this.ContentTextarea() : this.Placeholder())
			return views
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
			// After a successful save, update the baseline so the self-echo does not
			// trigger the "updated elsewhere" banner.
			for (const p of paths) {
				this.just_saved_path(p, Date.now())
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
				diff: this.Diff(),
			}
		}

		// Override the auto-generated state setter from the tree so that setting
		// a non-zero diff_from_version_id also opens the diff sidebar.
		override diff_from_version_id(next?: number): number {
			if (next !== undefined && next !== 0) {
				super.diff_from_version_id(next)
				this.right_sidebar('diff')
				return next
			}
			return super.diff_from_version_id(next)
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

		// ── Live update subscription ──────────────────────────────────────────────

		@$mol_mem
		subscription() {
			const path = this.path()
			if (!path) return null
			return $trip2g_graphql_raw_subscription(EDITOR_CHANGES_QUERY, {
				filter: { includePatterns: ['**/*.md'] },
			})
		}

		// Per-path record of the timestamp of the last local save — used to
		// suppress self-echo events that arrive shortly after we save.
		@$mol_mem_key
		just_saved_path(path: string, next?: number): number {
			return next ?? 0
		}

		// Whether the current path has a pending external update waiting for
		// the user's attention.
		@$mol_mem
		pending_external_update(next?: number | null): number | null {
			return next !== undefined ? next : null
		}

		// Reactive watcher: reads SSE data and sets pending_external_update when
		// the currently-open file is modified externally.
		@$mol_mem
		watcher_result(): null {
			const sub = this.subscription()
			if (!sub) return null

			const data = sub.data()
			if (!data) return null

			const path = this.path()
			if (!path) return null

			const changes: any[] = data.noteChanges?.changes ?? []
			for (const ch of changes) {
				if (ch.__typename === 'NoteUpsertEvent' && ch.path === path) {
					const savedAt = this.just_saved_path(path)
					// Suppress self-echo: ignore events within 5 seconds of a local save.
					if (savedAt && Date.now() - savedAt < 5000) continue
					const versionId: number = ch.versionId
					if (versionId) {
						this.pending_external_update(versionId)
					}
				}
			}

			return null
		}

		// ── Banner actions ────────────────────────────────────────────────────────

		override handle_show_diff(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				if (!path) return null
				// Fetch the two most recent version IDs for this path.
				const res = history_request({ filter: { path, limit: 2 } })
				const nodes = res.admin.noteVersionHistory.nodes
				if (nodes.length >= 2) {
					// nodes[0] is newest (highest version number), nodes[1] is previous
					this.diff_from_version_id(nodes[1].versionId)
					this.diff_to_version_id(nodes[0].versionId)
				} else if (nodes.length === 1) {
					this.diff_from_version_id(nodes[0].versionId)
					this.diff_to_version_id(nodes[0].versionId)
				}
				this.toggle_right_sidebar('diff')
			}
			return null
		}

		override handle_load_latest(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				if (!path) return null
				const hasChanges = this.changed_paths().includes(path)
				if (hasChanges) {
					if (!confirm('You have unsaved changes. Load the latest version and discard them?')) {
						return null
					}
					// Discard local changes for this path.
					this.change(path, null)
					this.changed_paths(this.changed_paths().filter(p => p !== path))
				}
				// Increment the reload counter to force loaded_note_path to re-fetch.
				this.reload_counter(path, this.reload_counter(path) + 1)
				this.pending_external_update(null)
				this.just_saved_path(path, Date.now())
			}
			return null
		}

		override handle_dismiss(next?: Event): null {
			if (next !== undefined) {
				this.pending_external_update(null)
			}
			return null
		}
	}
}
