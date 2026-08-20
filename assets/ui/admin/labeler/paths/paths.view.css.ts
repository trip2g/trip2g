namespace $ {
	const { rem } = $mol_style_unit

	// Note paths are long and there can be several per delivery, so the cell
	// wraps and breaks rather than pushing the row past the viewport.
	$mol_style_define( $trip2g_admin_labeler_paths, {
		flex: {
			basis: rem( 30 ),
		},
		minWidth: 0,
		Cell: {
			maxWidth: '100%',
			overflowWrap: 'anywhere',
		},
	} )
}
