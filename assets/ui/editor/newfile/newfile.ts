namespace $ {
	export type $trip2g_editor_newfile_result =
		| { readonly ok: true; readonly path: string }
		| { readonly ok: false; readonly error: 'empty' | 'exists' }

	// Normalise and validate a user-entered new-file path against the existing notes.
	// - trims surrounding whitespace and a leading slash
	// - empty input → { ok: false, error: 'empty' }
	// - appends `.md` when the name does not already end in `.md` (case-insensitive)
	// - a path that already exists (case-insensitive match) → { ok: false, error: 'exists' }
	// Pure: no DOM, network or state — unit-testable (see newfile.test.ts).
	export function $trip2g_editor_newfile_normalize(
		raw: string,
		existing: readonly string[],
	): $trip2g_editor_newfile_result {
		let path = raw.trim()
		if (path.startsWith('/')) path = path.slice(1)
		if (!path) return { ok: false, error: 'empty' }
		if (!path.toLowerCase().endsWith('.md')) path += '.md'

		const lower = path.toLowerCase()
		if (existing.some(p => p.toLowerCase() === lower)) {
			return { ok: false, error: 'exists' }
		}
		return { ok: true, path }
	}

	// Initial body for a freshly created note: an H1 derived from the file name
	// (basename without the `.md` extension).
	export function $trip2g_editor_newfile_initial_content(path: string): string {
		const name = (path.split('/').at(-1) ?? path).replace(/\.md$/i, '')
		return `# ${name}\n`
	}
}
