namespace $ {
	// Version-row label "v{version} · {date} · {size}b", or the raw id when the version is not
	// found. The date is passed in (locale-formatted by the caller) to keep this pure/testable.
	export function $trip2g_editor_versions_title(
		v: { version: number; contentLength: number } | undefined,
		id: string,
		date: string,
	): string {
		if (!v) return id
		return `v${v.version} · ${date} · ${v.contentLength}b`
	}

	// Given the version history (newest → oldest) and a selected version id, return the
	// { from (the next/older entry), to (selected) } id pair to diff — or null when the id is
	// missing or it is the oldest entry (nothing older to diff against).
	export function $trip2g_editor_versions_diff_pair(
		versions: readonly { versionId: number }[],
		id: string,
	): { from: number; to: number } | null {
		const idx = versions.findIndex(v => String(v.versionId) === id)
		if (idx < 0) return null
		if (idx + 1 >= versions.length) return null
		return { from: versions[idx + 1].versionId, to: versions[idx].versionId }
	}
}
