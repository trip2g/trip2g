namespace $ {
	// Returns the wikilink target whose [[…]] span contains the given caret offset, or
	// null if the offset is outside any. The alias (`|…`) and anchor (`#…`) suffixes are
	// stripped: [[Path/Note|Alias]] and [[Path/Note#Section]] both yield "Path/Note".
	export function $trip2g_editor_wikilink(text: string, offset: number): string | null {
		const re = /\[\[([^\]]+)\]\]/g
		let m: RegExpExecArray | null
		while ((m = re.exec(text)) !== null) {
			if (m.index <= offset && offset <= m.index + m[0].length) {
				return m[1].split('|')[0].split('#')[0].trim()
			}
		}
		return null
	}
}
