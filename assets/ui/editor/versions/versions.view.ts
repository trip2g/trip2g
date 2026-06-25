namespace $.$$ {
	const history_request = $trip2g_graphql_request(/* GraphQL */ `
		query EditorNoteVersions($filter: AdminNoteVersionHistoryFilter!) {
			admin {
				noteVersionHistory(filter: $filter) {
					nodes {
						versionId
						version
						contentLength
						createdAt
					}
				}
			}
		}
	`)

	const version_request = $trip2g_graphql_request(/* GraphQL */ `
		query EditorNoteVersion($versionId: Int64!) {
			admin {
				noteVersion(versionId: $versionId) {
					versionId
					content
				}
			}
		}
	`)

	type VersionMeta = { versionId: number; version: number; contentLength: number; createdAt: string }

	export class $trip2g_editor_versions extends $.$trip2g_editor_versions {
		@$mol_mem
		versions(): VersionMeta[] {
			const path = this.path()
			if (!path) return []
			const res = history_request({ filter: { path } })
			return res.admin.noteVersionHistory.nodes
		}

		override version_rows(): readonly $mol_view[] {
			return this.versions().map(v => this.Version(String(v.versionId)))
		}

		override version_title(id: string): string {
			const v = this.versions().find(x => String(x.versionId) === id)
			if (!v) return id
			const date = new Date(v.createdAt).toLocaleString()
			return `v${v.version} · ${date} · ${v.contentLength}b`
		}

		override version_click(id: string, next?: Event): null {
			if (next !== undefined) {
				const res = version_request({ versionId: Number(id) })
				this.content(res.admin.noteVersion?.content ?? '')
			}
			return null
		}

		override version_diff_click(id: string, next?: Event): null {
			if (next !== undefined) {
				const versions = this.versions()
				const idx = versions.findIndex(v => String(v.versionId) === id)
				if (idx < 0) return null
				const toVersionId = versions[idx].versionId
				// Previous version is the next one in the list (versions are desc by version number)
				const fromVersionId = idx + 1 < versions.length ? versions[idx + 1].versionId : toVersionId
				this.diff_from_version_id(fromVersionId)
				this.diff_to_version_id(toVersionId)
			}
			return null
		}
	}
}
