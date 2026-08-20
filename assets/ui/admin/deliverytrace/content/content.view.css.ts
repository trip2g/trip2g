namespace $ {
	const { rem } = $mol_style_unit

	// The stored bytes verbatim: wrapped, monospaced, never reflowed as prose.
	$mol_style_define( $trip2g_admin_deliverytrace_content, {
		padding: $mol_gap.text,
		maxHeight: rem( 24 ),
		overflow: 'auto',
		// The stored note is taller than one line: the expander's list would shrink
		// it to nothing otherwise.
		flex: { shrink: 0 },

		Text: {
			// display:block — a mol view is a flex container, and a bare text node in
			// one becomes an unwrappable flex item, so the note would show as a single
			// clipped line.
			display: 'block',
			whiteSpace: 'pre-wrap',
			overflowWrap: 'anywhere',
			font: { family: 'monospace', size: rem( 0.8 ) },
			color: $mol_theme.text,
		},
	} )
}
