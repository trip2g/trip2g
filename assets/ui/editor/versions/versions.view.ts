namespace $.$$ {
	type VersionMeta = { versionId: number; version: number; contentLength: number; createdAt: string }

	export class $trip2g_editor_versions extends $.$trip2g_editor_versions {
		@$mol_mem
		versions(): VersionMeta[] {
			const path = this.path()
			if (!path) return []
			const res = $trip2g_editor_versions_list({ filter: { path } })
			return res.admin.noteVersionHistory.nodes
		}

		override version_rows(): readonly $mol_view[] {
			return this.versions().map(v => this.Version(String(v.versionId)))
		}

		override version_title(id: string): string {
			const v = this.versions().find(x => String(x.versionId) === id)
			const date = v ? new Date(v.createdAt).toLocaleString() : ''
			return $trip2g_editor_versions_title(v, id, date)
		}

		override version_click(id: string, next?: Event): null {
			if (next !== undefined) {
				const res = $trip2g_editor_versions_get({ versionId: Number(id) })
				this.content(res.admin.noteVersion?.content ?? '')
			}
			return null
		}

		override version_diff_click(id: string, next?: Event): null {
			if (next !== undefined) {
				const pair = $trip2g_editor_versions_diff_pair(this.versions(), id)
				if (!pair) return null
				this.diff_from_version_id(pair.from)
				this.diff_to_version_id(pair.to)
				// Signal pane to open the diff sidebar via the show_diff? callback.
				this.show_diff(null)
			}
			return null
		}
	}
}
