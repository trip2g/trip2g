namespace $.$$ {
	export class $trip2g_admin_sync extends $.$trip2g_admin_sync {
		private _logLines: string[] = []
		private _uploadPaths: string[] = []
		private _autoTimer: ReturnType<typeof setInterval> | null = null

		// Stable, memoized CONSTRUCTION only (no $mol_wire_sync here — that rebuilt
		// the env on every restart → "Different host on restart" loop). init()
		// (restore a previously-granted folder) is fired-and-forgotten and bumps
		// _dir_ver when done.
		@$mol_mem
		env() {
			const env = new this.$.$trip2g_sync.BrowserEnv(
				{ apiUrl: '/_system/graphql', useCookieAuth: true, twoWaySync: false },
				{
					// Keep the activity log quiet: only surface warnings/errors, and
					// drop the "No stored directory…" notice — it's the normal state on
					// a fresh or just-cleared page, not something the user needs to see.
					onLog: (msg: string, level: string) => {
						if (msg.includes('No stored directory')) return
						if (level === 'warn' || level === 'error') this._appendLog(`[${level}] ${msg}`)
					},
				}
			)
			env.init()
				.then(() => {
					this._dir_ver(this._dir_ver() + 1)
					// A folder restored from IndexedDB on page reload also needs its
					// upload preview recomputed (otherwise it shows a stale "nothing
					// to upload" until the user re-picks the folder).
					this._refreshPlan()
				})
				.catch((e: unknown) => this._appendLog(`[error] init: ${(e as Error).message}`))
			return env
		}

		private _appendLog(line: string) {
			this._logLines = [...this._logLines.slice(-50), line]
			this._log_ver(this._log_ver() + 1)
		}

		private _clearLog() {
			this._logLines = []
			this._log_ver(this._log_ver() + 1)
		}

		// Counters bumped from async callbacks to force reactive re-reads.
		@$mol_mem _log_ver(next?: number): number { return next ?? 0 }
		@$mol_mem _dir_ver(next?: number): number { return next ?? 0 }
		@$mol_mem _plan_ver(next?: number): number { return next ?? 0 }

		@$mol_mem syncing(next?: boolean): boolean { return next ?? false }
		@$mol_mem sync_result_text(next?: string): string { return next ?? '' }

		// Preview: which files WOULD be uploaded, without sending anything.
		private _refreshPlan() {
			if (!this.env().isReady()) {
				this._uploadPaths = []
				this._plan_ver(this._plan_ver() + 1)
				return
			}
			this.env().getSyncPlan()
				.then(plan => {
					this._uploadPaths = [...plan.pushes, ...plan.localOnly].map(f => f.path).sort()
					this._plan_ver(this._plan_ver() + 1)
				})
				.catch((e: unknown) => this._appendLog(`[error] preview: ${(e as Error).message}`))
		}

		override log_text(): string {
			this._log_ver()
			return this._logLines.join('\n')
		}

		override status_text(): string {
			this._dir_ver()
			const name = this.env().getDirectoryName()
			return name ? `Folder: ${name}` : 'No folder selected'
		}

		override upload_summary(): string {
			this._plan_ver()
			this._dir_ver()
			if (!this.env().isReady()) return ''
			const n = this._uploadPaths.length
			return n === 0 ? 'Up to date — nothing to upload' : `Will upload ${n} file(s):`
		}

		@$mol_mem
		override upload_rows(): readonly $mol_view[] {
			this._plan_ver()
			return this._uploadPaths.map(p => this.Row(p))
		}

		// Keyed by the file path, so $mol reuses row views stably across refreshes.
		override upload_path(id: string): string {
			return id
		}

		override folder_ready(): boolean {
			this._dir_ver()
			return this.env().isReady()
		}

		override refresh(event?: null) {
			this._refreshPlan()
		}

		override sync_disabled(): boolean {
			this._dir_ver()
			this._plan_ver()
			return this.syncing() || !this.env().isReady() || this._uploadPaths.length === 0
		}

		// Show Upload / Clear only when a folder is selected; hidden (null) otherwise.
		override upload_button() {
			this._dir_ver()
			return this.env().isReady() ? this.UploadButton() : null
		}

		override clear_folder_button() {
			this._dir_ver()
			return this.env().isReady() ? this.ClearFolder() : null
		}

		override sync_result(): string {
			return this.sync_result_text()
		}

		@$mol_mem
		override auto_sync_controls(): $mol_view[] {
			this._dir_ver()
			return this.env().isReady() ? [this.AutoSyncCheck()] : []
		}

		@$mol_mem
		override auto_sync(next?: boolean): boolean {
			const key = 'trip2g_admin_sync_autosync'
			if (next !== undefined) {
				localStorage.setItem(key, next ? '1' : '0')
				if (next) this._startAutoSync()
				else this._stopAutoSync()
				return next
			}
			return localStorage.getItem(key) === '1'
		}

		private _startAutoSync() {
			if (this._autoTimer) return
			this._autoTimer = setInterval(() => {
				if (this.syncing()) return
				if (!this.env().isReady()) return
				this.sync_now(null)
			}, 30_000)
		}

		private _stopAutoSync() {
			if (this._autoTimer) {
				clearInterval(this._autoTimer)
				this._autoTimer = null
			}
		}

		override select_dir(event?: null) {
			this._clearLog()
			// Direct call (no $mol_wire_sync) keeps the user gesture for the picker.
			this.env().selectDirectory()
				.then(ok => {
					if (!ok) this._appendLog('[warn] no folder selected')
					this._dir_ver(this._dir_ver() + 1)
					this._refreshPlan()
				})
				.catch((e: unknown) => this._appendLog(`[error] select: ${(e as Error).message}`))
		}

		override sync_now(event?: null) {
			if (this.syncing()) return
			this.syncing(true)
			this.sync_result_text('')
			this.env().sync()
				.then(r => {
					this.sync_result_text(
						`Uploaded ${r.pushed} file(s)` + (r.errors.length ? `, ${r.errors.length} error(s)` : '')
					)
				})
				.catch((e: unknown) => this._appendLog(`[error] sync: ${(e as Error).message}`))
				.finally(() => {
					this.syncing(false)
					this._dir_ver(this._dir_ver() + 1)
					this._refreshPlan()
				})
		}

		override clear_dir(event?: null) {
			this.env().clearDirectory()
				.then(() => {
					this._uploadPaths = []
					this.sync_result_text('')
					this._clearLog()
					this._plan_ver(this._plan_ver() + 1)
					this._dir_ver(this._dir_ver() + 1)
				})
				.catch((e: unknown) => this._appendLog(`[error] clear: ${(e as Error).message}`))
		}
	}
}
