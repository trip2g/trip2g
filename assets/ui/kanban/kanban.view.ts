namespace $.$$ {
	type Spec = { path: string; versionId: number; editable: boolean; markdown: string }
	type TransferData = { from: number; card: number }

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
						id
					}
				}
			}
		}
	`)

	export class $trip2g_kanban extends $.$trip2g_kanban {

		// --- server data island ---

		@$mol_mem
		spec(): Spec {
			const el = document.getElementById('kanban-data')
			if (!el?.textContent) return { path: '', versionId: 0, editable: false, markdown: '' }
			return JSON.parse(el.textContent) as Spec
		}

		override editable(): boolean {
			return this.spec().editable
		}

		// --- board model (reactive store) ---

		@$mol_mem
		board(next?: $trip2g_kanban_board): $trip2g_kanban_board {
			if (next !== undefined) return next
			return $trip2g_kanban_parse_board(this.spec().markdown)
		}

		// Tracks the last saved version id for SSE echo suppression.
		@$mol_mem
		baseline_version(next?: number): number {
			return next !== undefined ? next : this.spec().versionId
		}

		// --- columns ---

		override columns(): readonly $mol_view[] {
			return this.board().lists.map((_, i) => this.Column(i))
		}

		override column_title(id: number): string {
			return this.board().lists[id]?.title ?? ''
		}

		override card_rows(id: number): readonly $mol_view[] {
			return (this.board().lists[id]?.cards ?? []).map((_, j) => this.Card(`${id}:${j}`))
		}

		// --- cards (keyed "listIndex:cardIndex") ---

		private card_idx(id: string): [number, number] {
			const [l, c] = id.split(':').map(Number)
			return [l, c]
		}

		override card_json(id: string): string {
			const [l, c] = this.card_idx(id)
			return JSON.stringify({ from: l, card: c })
		}

		override card_checked(id: string, val?: boolean): boolean {
			const [l, c] = this.card_idx(id)
			const current = this.board().lists[l]?.cards[c]?.checked ?? false
			if (val !== undefined && val !== current) {
				if (!this.editable()) return current
				this.commit($trip2g_kanban_edit_card(this.board(), l, c, { checked: val }))
			}
			return this.board().lists[l]?.cards[c]?.checked ?? false
		}

		// In-progress draft per card: null means no active edit (use board value).
		@$mol_mem_key
		card_text_draft(id: string, next?: string | null): string | null {
			if (next !== undefined) return next
			return null
		}

		override card_text(id: string, val?: string): string {
			const [l, c] = this.card_idx(id)
			// Always read board so it stays a tracked dependency — ensures
			// that board changes (SSE updates, commits) invalidate this memo.
			const boardText = this.board().lists[l]?.cards[c]?.text ?? ''
			if (val !== undefined) {
				if (this.editable()) {
					this.card_text_draft(id, val)
				}
				return val
			}
			const draft = this.card_text_draft(id)
			return draft !== null ? draft : boardText
		}

		// Commit the draft when the text field loses focus.
		// Called via the DOM `focusout` event wired in kanban.view.tree.
		// Runs inside $mol_wire_async so reactive setters are safe to call.
		card_text_focusout(id: string, _event?: Event): void {
			const [l, c] = this.card_idx(id)
			const draft = this.card_text_draft(id)
			if (draft === null) return
			const board = this.board()
			const card = board.lists[l]?.cards[c]
			if (!card || draft === card.text) {
				this.card_text_draft(id, null)
				return
			}
			this.commit($trip2g_kanban_edit_card(board, l, c, { text: draft }))
			this.card_text_draft(id, null)
		}

		// --- drag / drop ---

		card_adopt(transfer: DataTransfer): TransferData | null {
			const json = transfer.getData('text/plain')
			if (!json) return null
			try { return JSON.parse(json) as TransferData } catch { return null }
		}

		override card_receive(id: number, obj?: TransferData): void {
			if (!obj) return
			if (!this.editable()) return
			this.commit($trip2g_kanban_move_card(this.board(), {
				from: obj.from,
				card: obj.card,
				to: id,
				at: this.board().lists[id]?.cards.length ?? 0,
			}))
		}

		// --- add card ---

		override add_card(id: number, event?: null): void {
			if (!this.editable()) return
			const text = window.prompt('Card text', '')
			if (text !== null && text !== '') {
				this.commit($trip2g_kanban_add_card(this.board(), id, text))
			}
		}

		// --- persistence ---

		private commit(board: $trip2g_kanban_board): void {
			this.board(board)
			this.save(board)
		}

		private save(board: $trip2g_kanban_board): void {
			const content = $trip2g_kanban_serialize_board(board)
			const res = save_mutate({ input: { updates: [{ path: this.spec().path, content }] } })
			if (res.pushNotes.__typename === 'ErrorPayload') {
				throw new Error(res.pushNotes.message)
			}
			// Advance the baseline to suppress the self-echo SSE event from the save.
			const updated = (res.pushNotes as any).updated as Array<{ path: string; id: number }> | undefined
			for (const u of updated ?? []) {
				if (u.id) this.baseline_version(Number(u.id))
			}
		}
	}
}
