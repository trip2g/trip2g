namespace $ {
	const { rem } = $mol_style_unit

	// The agent's own bag, verbatim: wrapped, monospaced, never reflowed as prose.
	// Mirrors the note-content viewer next door, for the same reasons.
	$mol_style_define( $trip2g_admin_deliverytrace_log_data, {
		padding: $mol_gap.text,
		maxHeight: rem( 24 ),
		overflow: 'auto',
		flex: { shrink: 0 },

		Text: {
			display: 'block',
			whiteSpace: 'pre-wrap',
			overflowWrap: 'anywhere',
			font: { family: 'monospace', size: rem( 0.8 ) },
			color: $mol_theme.text,
		},
	} )

	// The one line every agent means the same thing by: time, level, message.
	$mol_style_define( $trip2g_admin_deliverytrace_log, {
		Entry_line: {
			font: { family: 'monospace', size: rem( 0.8 ) },
			whiteSpace: 'pre-wrap',
			overflowWrap: 'anywhere',
		},
	} )
}
