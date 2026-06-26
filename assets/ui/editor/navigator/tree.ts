namespace $ {
	// Case-insensitive substring filter over note paths. An empty/whitespace query keeps
	// every path (order preserved); otherwise keeps paths containing the trimmed query.
	export function $trip2g_editor_navigator_filter(paths: readonly string[], query: string): string[] {
		const q = query.trim().toLowerCase()
		const out: string[] = []
		for (const path of paths) {
			if (q && !path.toLowerCase().includes(q)) continue
			out.push(path)
		}
		return out
	}

	// Keep the ids whose any tag lives under the folder prefix. Root ('') keeps all; a tag must
	// start with `prefix + '/'`, so a tag equal to the prefix (no trailing slash) is excluded.
	export function $trip2g_editor_navigator_ids(
		ids_tags: Record<string, readonly string[]>,
		prefix: string,
	): string[] {
		return Object.keys(ids_tags).filter(id =>
			ids_tags[id].some(tag => prefix === '' || tag.startsWith(prefix + '/')),
		)
	}

	// Immediate subfolder names directly under the prefix, deduped and sorted. Each tag is taken
	// relative to the prefix; only tags with a further '/' contribute a folder (leaf files don't).
	export function $trip2g_editor_navigator_subfolders(
		ids: readonly string[],
		ids_tags: Record<string, readonly string[]>,
		prefix: string,
		cmp: (a: string, b: string) => number = (a, b) => (a < b ? -1 : a > b ? 1 : 0),
	): string[] {
		const folders = new Set<string>()
		for (const id of ids) {
			for (const tag of ids_tags[id] ?? []) {
				const rest = prefix ? tag.slice(prefix.length + 1) : tag
				const slash = rest.indexOf('/')
				if (slash >= 0) folders.add(rest.slice(0, slash))
			}
		}
		return [...folders].sort(cmp)
	}
}
