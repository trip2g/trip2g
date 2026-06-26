namespace $ {
	// Classify a unified-diff line for styling: an added '+' line (but not the '+++'
	// file header) → 'diff-add', a removed '-' line (but not '---') → 'diff-del',
	// anything else (context, hunk header '@@', empty) → ''.
	export function $trip2g_editor_diff_line_class(line: string): string {
		if (line.startsWith('+') && !line.startsWith('+++')) return 'diff-add'
		if (line.startsWith('-') && !line.startsWith('---')) return 'diff-del'
		return ''
	}
}
