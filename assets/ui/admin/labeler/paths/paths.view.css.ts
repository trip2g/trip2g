namespace $ {
	const { rem } = $mol_style_unit

	// The cell is allowed to shrink and clip: a note path is long enough to push
	// the whole row past the pane otherwise, and the paths inside it ellipsise in
	// the middle rather than the row growing to fit them.
	$mol_style_define( $trip2g_admin_labeler_paths, {
		flex: {
			basis: rem( 30 ),
			grow: 1,
			shrink: 1,
		},
		minWidth: 0,
		// In a row the basis above decides the width; in the vertical details pane a
		// flex-basis means height, so the cap has to be a width of its own.
		width: '100%',
		// A hard cap, not 100%: the pane sizes itself to its content, so a percentage
		// resolves in a circle and the whole book grows wider than the window.
		maxWidth: rem( 24 ),
		overflow: 'hidden',

		Cell: {
			padding: $mol_gap.text,
			gap: rem( 0.25 ),
			minWidth: 0,
			overflow: 'hidden',
		},
	} )
}
