namespace $.$$ {
	// Viewer role (admin gate) + the authoritative DOM-index → source-line
	// mapping extracted server-side from the AST. Re-fetched after mutations
	// (updateNotes reloads NoteViews synchronously, so the refetch is fresh).
	const tasklist_request = $trip2g_graphql_request(/* GraphQL */ `
		query NoteTaskList($input: NoteInput!) {
			viewer {
				role
			}
			note(input: $input) {
				tasklist {
					index
					line
					checked
					text
				}
			}
		}
	`)

	const content_request = $trip2g_graphql_request(/* GraphQL */ `
		query TaskListNoteContent($filter: NotePathsFilter) {
			notePaths(filter: $filter) {
				id
				content
			}
		}
	`)

	const save_mutate = $trip2g_graphql_request(/* GraphQL */ `
		mutation TaskListToggle($input: UpdateNotesInput!) {
			updateNotes(input: $input) {
				__typename
				... on UpdateNotesSuccessPayload {
					updated {
						path
						versionId
					}
				}
				... on UpdateNotesHashMismatchPayload {
					path
					actualHash
				}
				... on UpdateNotesPatchNotFoundPayload {
					path
					find
				}
				... on ErrorPayload {
					message
				}
			}
		}
	`)

	type TaskItem = {
		index: number
		line: number
		checked: boolean
		text: string
	}

	// Flips the task marker in a source line: "[ ]" ↔ "[x]"/"[X]". The marker
	// is the first occurrence matching the current checked state — anything
	// before it can only be list/quote prefixes. Returns null when not found.
	function toggle_marker(line: string, checked: boolean): string | null {
		const markers = checked ? ['[x]', '[X]'] : ['[ ]']

		let pos = -1
		for (const marker of markers) {
			const at = line.indexOf(marker)
			if (at >= 0 && (pos < 0 || at < pos)) pos = at
		}
		if (pos < 0) return null

		return line.slice(0, pos) + (checked ? '[ ]' : '[x]') + line.slice(pos + 3)
	}

	function count_occurrences(haystack: string, needle: string): number {
		let count = 0
		for (let at = haystack.indexOf(needle); at >= 0; at = haystack.indexOf(needle, at + 1)) {
			count++
			if (count > 1) break
		}
		return count
	}

	// Same content hash as the server's updatenotes.hashContent:
	// base64url(sha256(content)), padding kept.
	function content_hash(content: string): string {
		const bytes = new TextEncoder().encode(content)
		const digest = $mol_wire_sync(crypto.subtle).digest('SHA-256', bytes) as ArrayBuffer
		let bin = ''
		for (const byte of new Uint8Array(digest)) bin += String.fromCharCode(byte)
		return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_')
	}

	export class $trip2g_user_tasklist extends $.$trip2g_user_tasklist {
		@$mol_mem
		path_id() {
			const el = this.dom_node() as HTMLElement
			if (!el.dataset.pid) throw new Error('pid not found in dataset')
			return parseInt(el.dataset.pid, 10)
		}

		@$mol_mem
		note_path() {
			const el = this.dom_node() as HTMLElement
			if (!el.dataset.path) throw new Error('path not found in dataset')
			return el.dataset.path
		}

		@$mol_mem
		snapshot() {
			return tasklist_request({
				input: { pathId: this.path_id(), referer: this.$.$mol_dom_context.location.pathname },
			})
		}

		is_admin() {
			return this.snapshot().viewer.role === 'ADMIN'
		}

		items(): readonly TaskItem[] {
			return this.snapshot().note?.tasklist ?? []
		}

		// Version id of our last own save, advanced from updated[].versionId
		// (kanban self-echo pattern: events at or below it are our own writes).
		@$mol_mem
		baseline_version_id(next?: number) {
			return next ?? 0
		}

		@$mol_mem
		error_message(next?: string) {
			return next ?? ''
		}

		// The note-body checkboxes rendered by goldmark, in document order.
		// Scoped to the closest article so embedded ref-notes don't interfere.
		@$mol_mem
		checkboxes(): HTMLInputElement[] {
			const el = this.dom_node() as HTMLElement
			const body = el.closest('article')?.querySelector('.content__body')
			if (!body) return []
			return Array.from(body.querySelectorAll('li input[type="checkbox"]'))
		}

		// Enables the checkboxes and binds click handlers (once per element).
		// Safety: if the DOM checkbox count differs from the extracted task
		// count the index mapping cannot be trusted — leave everything
		// read-only rather than risk patching the wrong source line.
		@$mol_mem
		bound(): boolean {
			if (!this.is_admin()) return false

			const boxes = this.checkboxes()
			if (boxes.length === 0 || boxes.length !== this.items().length) return false

			boxes.forEach((box, index) => {
				box.disabled = false
				if (box.dataset.tasklistBound) return
				box.dataset.tasklistBound = '1'
				box.addEventListener('click', (event: Event) => {
					$mol_wire_async(this).toggle(index, event)
				})
			})

			return true
		}

		toggle(index: number, event: Event) {
			const box = event.target as HTMLInputElement
			const next_checked = box.checked // browser already flipped it
			const item = this.items()[index]
			if (!item || item.checked === next_checked) return

			const boxes = this.checkboxes()
			try {
				boxes.forEach(other => { other.disabled = true })
				this.save(item, next_checked)
				this.error_message('')
			} catch (error) {
				if ($mol_promise_like(error)) $mol_fail_hidden(error)
				box.checked = !next_checked // revert
				this.error_message((error as Error).message || 'Failed to save task')
			} finally {
				boxes.forEach(other => { other.disabled = false })
			}
		}

		save(item: TaskItem, next_checked: boolean) {
			const path = this.note_path()

			const res = content_request({ filter: { paths: [path] } })
			const content: string | undefined = res.notePaths[0]?.content
			if (content == null) throw new Error('Note content unavailable')

			const lines = content.split('\n')
			const raw_line = lines[item.line - 1] ?? ''
			if (raw_line.replace(/\r$/, '') !== item.text) {
				throw new Error('Note changed since page load — reload the page')
			}

			const new_line = toggle_marker(item.text, item.checked)
			if (new_line == null) throw new Error('Task marker not found in source line')

			let change
			if (count_occurrences(content, item.text) === 1) {
				change = { patch: { path, find: item.text, replace: new_line } }
			} else {
				// The line is not unique — compare-and-swap the whole note.
				lines[item.line - 1] = new_line + (raw_line.endsWith('\r') ? '\r' : '')
				change = { upsert: { path, content: lines.join('\n'), expectedHash: content_hash(content) } }
			}

			const result = save_mutate({ input: { changes: [change] } })
			const payload = result.updateNotes

			switch (payload.__typename) {
				case 'UpdateNotesSuccessPayload': {
					const version_id = Number(payload.updated[0]?.versionId ?? 0)
					if (version_id) this.baseline_version_id(version_id)
					return
				}
				case 'UpdateNotesHashMismatchPayload':
				case 'UpdateNotesPatchNotFoundPayload':
					throw new Error('Note changed since page load — reload the page')
				case 'ErrorPayload':
					throw new Error(payload.message)
				default:
					throw new Error('Unexpected save response')
			}
		}

		override sub() {
			this.bound()
			if (!this.error_message()) return []
			return [this.Error()]
		}
	}
}
