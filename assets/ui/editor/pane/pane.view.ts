namespace $.$$ {
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
		loaded_note_path(path: string): { id: any; content: string; versionId?: number } | null {
			if (!path) return null
			this.reload_counter(path) // reactive dependency
			const res = $trip2g_editor_pane_content({ filter: { paths: [path] } })
			const np = res.notePaths[0]
			if (!np) return null
			// Initialise baseline from the freshly loaded version (if available).
			// Use the version history to get the latest versionId for this path.
			const hist = $trip2g_editor_pane_history({ filter: { path, limit: 1 } })
			const latestVersionId = hist.admin.noteVersionHistory.nodes[0]?.versionId ?? 0
			if (latestVersionId) {
				setTimeout(() => { this.baseline_version_id(path, latestVersionId) }, 0)
			}
			return { id: np.id, content: np.content }
		}

		@$mol_mem_key
		loaded_content(path: string): string {
			return this.loaded_note_path(path)?.content ?? ''
		}

		note_path_id(): any {
			return this.loaded_note_path(this.path())?.id ?? null
		}

		// Version-ID baseline per path: events with versionId <= baseline are self-echoes.
		@$mol_mem_key
		baseline_version_id(path: string, next?: number): number {
			return next ?? 0
		}

		override handle_content_click(next?: MouseEvent | null): null {
			if (next?.ctrlKey) {
				const pos = document.caretPositionFromPoint(next.clientX, next.clientY)
				if (pos?.offsetNode?.nodeType === Node.TEXT_NODE) {
					const link = $trip2g_editor_wikilink(this.content(), pos.offset)
					if (link) {
						const pathId = this.note_path_id()
						if (pathId !== null) {
							const res = $trip2g_editor_pane_resolve({ filter: { notePathId: pathId, links: [link] } })
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
						? $trip2g_editor_wikilink(this.content(), pos.offset)
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
			const res = $trip2g_editor_pane_save({ input: { updates } })
			if (res.pushNotes.__typename === 'ErrorPayload') {
				throw new Error(res.pushNotes.message)
			}
			// Advance the baseline to the just-saved version id, taken directly from the
			// mutation response (a re-fetch returns a stale cached version). This suppresses
			// the self-echo SSE event so our own save is not shown as "updated elsewhere".
			const updated = res.pushNotes.updated
			for (const u of updated ?? []) {
				if (u.id) this.baseline_version_id(u.path, Number(u.id))
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
				newfile: this.NewFile(),
			}
		}

		override handle_newfile_click() {
			this.newfile_error('')
			this.toggle_right_sidebar('newfile')
		}

		// Create a brand-new note via a create-only upsert (updateNotes, expectedHash:"").
		// The client-side paths() check below is a fast UX pre-reject; the server's
		// expectedHash:"" is the authoritative, race-proof guard behind it (it refuses to
		// overwrite a note created concurrently, e.g. via Obsidian sync, after the check).
		override handle_create_file(next?: Event): null {
			if (next === undefined) return null
			const res = $trip2g_editor_newfile_normalize(this.newfile_value(), this.Navigator().paths())
			if (!res.ok) {
				this.newfile_error(res.error === 'exists' ? this.newfile_msg_exists() : this.newfile_msg_empty())
				return null
			}
			const content = $trip2g_editor_newfile_initial_content(res.path)
			const result = $trip2g_editor_pane_create({
				input: { changes: [{ upsert: { path: res.path, content, expectedHash: '' } }] },
			})
			const payload = result.updateNotes
			if (payload.__typename === 'UpdateNotesHashMismatchPayload') {
				// The note already exists server-side — created concurrently after our
				// client-side paths() check passed. Refuse and tell the user.
				this.newfile_error(this.newfile_msg_exists_remote())
				return null
			}
			if (payload.__typename === 'ErrorPayload') {
				this.newfile_error(payload.message)
				return null
			}
			// UpdateNotesSuccessPayload: the note was created. updateNotes returns only
			// paths (no version id), so fetch the just-created version id and advance the
			// baseline synchronously — this suppresses the create's own self-echo SSE event
			// before the file is opened.
			const hist = $trip2g_editor_pane_history({ filter: { path: res.path, limit: 1 } })
			const latestVersionId = hist.admin.noteVersionHistory.nodes[0]?.versionId ?? 0
			if (latestVersionId) this.baseline_version_id(res.path, Number(latestVersionId))
			this.newfile_value('')
			this.newfile_error('')
			this.right_sidebar('')
			// Open the new note: select it and force a fresh load.
			this.path(res.path)
			this.reload_counter(res.path, this.reload_counter(res.path) + 1)
			return null
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

		@$mol_mem
		subscription() {
			const path = this.path()
			if (!path) return null
			// Use the shared cached host (same proven pattern as user/live): one stable
			// stream per query+vars, so re-evaluating this getter does not abort it.
			return $trip2g_graphql_raw_subscription(EDITOR_CHANGES_QUERY, {
				filter: { includePatterns: ['**/*.md'] },
			})
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

			const err = sub.error()
			if (err) console.log('editor subscription error', err)

			const data = sub.data()
			if (!data) return null

			const path = this.path()
			if (!path) return null

			// Match incoming events by note path id (robust), like user/live does —
			// the raw path string can differ from the editor's open-file path.
			const currentPathId = this.note_path_id()
			const changes: any[] = data.noteChanges?.changes ?? []
			for (const ch of changes) {
				if (ch.__typename !== 'NoteUpsertEvent') continue
				if (String(ch.pathId) !== String(currentPathId)) continue
				const versionId: number = ch.versionId
				if (!versionId) continue
				// Suppress self-echo: ignore events at or below the baseline version.
				if (versionId <= this.baseline_version_id(path)) continue
				// Defer state write to avoid synchronous mol memo mutation.
				setTimeout(() => { this.pending_external_update(versionId) }, 0)
			}

			return null
		}

		override handle_show_diff(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				if (!path) return null
				// Fetch the two most recent version IDs for this path.
				const res = $trip2g_editor_pane_history({ filter: { path, limit: 2 } })
				const nodes = res.admin.noteVersionHistory.nodes
				// Need at least 2 versions to show a meaningful diff.
				if (nodes.length < 2) return null
				// nodes[0] is newest (highest version number), nodes[1] is previous
				this.diff_from_version_id(nodes[1].versionId)
				this.diff_to_version_id(nodes[0].versionId)
				this.right_sidebar('diff')
			}
			return null
		}

		// Called when Versions panel triggers a diff via its show_diff? callback.
		override handle_versions_show_diff(next?: Event | null): null {
			if (next !== undefined) {
				this.right_sidebar('diff')
			}
			return null
		}

		override handle_load_latest(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				if (!path) return null
				const versionId = this.pending_external_update()
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
				// Advance the baseline synchronously: the reload re-evaluates the watcher
				// against the still-cached SSE event, which would otherwise re-raise the banner.
				if (versionId) this.baseline_version_id(path, versionId)
				this.pending_external_update(null)
			}
			return null
		}

		override handle_keep_mine(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				const versionId = this.pending_external_update()
				if (path && versionId) this.baseline_version_id(path, versionId)
				this.pending_external_update(null)
			}
			return null
		}

		// Keep both versions: drop git-style conflict markers (mine + the external
		// version, fetched by its exact versionId) into the editor for manual merge.
		override handle_keep_both(next?: Event): null {
			if (next !== undefined) {
				const path = this.path()
				const versionId = this.pending_external_update()
				if (!path || !versionId) return null
				const res = $trip2g_editor_pane_version({ versionId })
				const theirs = res.admin.noteVersion?.content ?? ''
				const mine = this.content()
				const merged = $trip2g_editor_merge(mine, theirs)
				this.content(merged)
				this.baseline_version_id(path, versionId)
				this.pending_external_update(null)
			}
			return null
		}
	}
}
