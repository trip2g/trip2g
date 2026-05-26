namespace $.$$ {
	// Capture this bundle's own URL at module-load time (currentScript is the
	// executing <script>). mol removes the <script> from the DOM after render,
	// so this can't be read later. Works in both the served app and the dev page.
	const bundle_url = (() => {
		const cs = $mol_dom_context.document.currentScript as HTMLScriptElement | null
		return cs?.src ?? ''
	})()

	export class $trip2g_editor extends $.$trip2g_editor {
		@$mol_mem
		listen() {
			this.$.$mol_dom_context.addEventListener('message', (e: MessageEvent) => {
				if (e.data === 'trip2g_editor_close') this.close()
				else if (e.data && e.data.type === 'trip2g_editor_path') {
					this.$.$mol_state_arg.value('editor_path', e.data.path || null)
				}
			})
			return true
		}

		// Read once, non-reactively, so srcdoc never rebuilds (which would reload the iframe).
		initial_path(): string {
			const hash = this.$.$mol_dom_context.location.hash
			const m = hash.match(/editor_path=([^/]*)/)
			const p = m ? decodeURIComponent(m[1]) : ''
			return p || $trip2g_settings.note_path()
		}

		@$mol_mem
		override srcdoc(): string {
			this.listen()
			const inner = {
				ui_lang: $trip2g_settings.ui_lang(),
				note_lang: $trip2g_settings.note_lang(),
				note_path: this.initial_path(),
				note_path_id: $trip2g_settings.note_path_id(),
				note_version_id: $trip2g_settings.note_version_id(),
			}
			const scripts = bundle_url ? `<script src="${bundle_url}" defer></script>` : ''
			return (
				'<!doctype html><html><head><meta charset="utf-8">' +
				`<script>window.__trip2g_settings=${JSON.stringify(inner)}</script>` +
				'</head><body style="margin:0;height:100vh"><div mol_view_root="$trip2g_editor_pane" style="height:100vh"></div>' +
				scripts +
				'</body></html>'
			)
		}

		dialog_dom() {
			return this.Dialog().dom_node() as HTMLDialogElement
		}

		_mounted = false

		open_status(opened?: boolean): string | null {
			const KEY = 'editor'

			setTimeout(() => {
				if (this.$.$mol_state_arg.value(KEY) === 'open') {
					this.dialog_dom().showModal()
				} else {
					this.dialog_dom().close()
				}

				this._mounted = true
			}, 10)

			if (opened !== undefined) {
				const newVal = opened ? 'open' : null
				this.$.$mol_state_arg.value(KEY, newVal)
				return newVal
			} else {
				const stateOpened = this.$.$mol_state_arg.value(KEY) === 'open'

				if (this._mounted) {
					if (stateOpened) {
						this.dialog_dom().showModal()
					} else {
						this.dialog_dom().close()
					}
				}
			}

			return opened ? 'open' : null
		}

		open() {
			this.open_status(true)
		}

		close() {
			this.open_status(false)
		}

		override close_event() {
			this.open_status(false)
		}

		override opened(next?: boolean) {
			if (next !== undefined) {
				this.open_status(next)
			}

			return this.$.$mol_state_arg.value('editor') === 'open'
		}
	}
}
